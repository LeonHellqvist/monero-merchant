package misc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/monero-merchant/monero-merchant/backend/internal/thirdparty/moneropay"
)

type MiscHandler struct {
	service *MiscService
}

func NewMiscHandler(service *MiscService) *MiscHandler {
	return &MiscHandler{service: service}
}

type Services struct {
	Postgresql bool                     `json:"postgresql"`
	MoneroPay  moneropay.HealthResponse `json:"MoneroPay"`
}

type HealthResponse struct {
	Status   int      `json:"status"`
	Services Services `json:"services"`
}

func (h *MiscHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	resp := h.service.GetHealth(ctx)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	io.Copy(io.Discard, r.Body)
}

type DaemonStatusResponse struct {
	Status              int    `json:"status"`
	Synced              bool   `json:"synced"`
	Height              uint64 `json:"height"`
	TargetHeight        uint64 `json:"target_height"`
	Nettype             string `json:"nettype"`
	IncomingConnections uint64 `json:"incoming_connections"`
	OutgoingConnections uint64 `json:"outgoing_connections"`
}

func (h *MiscHandler) GetDaemonStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := h.service.GetDaemonStatus(ctx)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
