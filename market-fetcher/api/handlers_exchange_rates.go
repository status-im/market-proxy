package api

import (
	"net/http"
)

// handleExchangeRates serves the cached CoinGecko /api/v3/exchange_rates body
// verbatim (Passthrough). Currency conversion does not depend on it.
func (s *Server) handleExchangeRates(w http.ResponseWriter, r *http.Request) {
	if s.exchangeRatesService == nil {
		s.sendJSONError(w, http.StatusServiceUnavailable, "exchange rates service is not available")
		return
	}

	rates, err := s.exchangeRatesService.ExchangeRates()
	if err != nil {
		s.sendJSONError(w, http.StatusServiceUnavailable, "Failed to get exchange rates: "+err.Error())
		return
	}

	s.sendJSONResponse(w, rates)
}
