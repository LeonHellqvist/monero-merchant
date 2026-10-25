package rates

import (
	"encoding/json"
	"errors"
	"net/http"
)

type RatesHandler struct {
	service *RatesService
}

func NewRatesHandler(service *RatesService) *RatesHandler {
	return &RatesHandler{service: service}
}

func (h *RatesHandler) GetRates(w http.ResponseWriter, r *http.Request) {
	currencies, err := ParseCurrencies(r.URL.Query().Get("currencies"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetRates(r.Context(), currencies)
	if errors.Is(err, ErrUnavailable) {
		http.Error(w, "Exchange rates are not available yet", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		http.Error(w, "Failed to get exchange rates", http.StatusInternalServerError)
		return
	}

	writeJSON(w, resp)
}

type currenciesResponse struct {
	Currencies []string `json:"currencies"`
}

func (h *RatesHandler) GetCurrencies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, currenciesResponse{Currencies: SupportedCurrencies})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
