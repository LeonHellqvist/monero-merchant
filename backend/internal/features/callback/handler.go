package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/monero-merchant/monero-merchant/backend/internal/core/utils"
)

type CallbackHandler struct {
	service *CallbackService
}

func NewCallbackHandler(service *CallbackService) *CallbackHandler {
	return &CallbackHandler{service: service}
}

func (h *CallbackHandler) ReceiveTransaction(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	jwtToken := chi.URLParam(r, "jwt")

	if err := h.service.HandleCallback(ctx, jwtToken); err != nil {
		http.Error(w, "callback handling failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	resp := "OK"
	_ = json.NewEncoder(w).Encode(resp)
	io.Copy(io.Discard, r.Body)
}

// Monero Light Wallet Server webhook
func (h *CallbackHandler) LwsHook(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	if !utils.SecureCompare(chi.URLParam(r, "jwt"), h.service.config.JWTLwsToken) {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("lws-hook: failed reading body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("lws-hook: payload: %s", string(body))

	var req LwsHookRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		log.Printf("lws-hook: decode error: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.HandleLwsHook(ctx, req); err != nil {
		http.Error(w, err.Message, err.Code)
		return
	}

	w.WriteHeader(http.StatusOK)
	resp := "OK"
	_ = json.NewEncoder(w).Encode(resp)
	io.Copy(io.Discard, r.Body)
}
