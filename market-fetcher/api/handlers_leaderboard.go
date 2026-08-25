package api

import (
	"fmt"
	"net/http"

	"github.com/status-im/market-proxy/interfaces"
)

// convertCurrencyParam is the optional query parameter requesting a realtime
// Estimate in another currency. It is deliberately separate from `currency`,
// which keeps its Passthrough-selection semantics.
const convertCurrencyParam = "convert_currency"

// emptyMarketsResponse is the response shape used when no markets data is available
func emptyMarketsResponse() map[string]interface{} {
	return map[string]interface{}{
		"data": []interface{}{},
	}
}

// resolveConvertCurrency parses and validates the convert_currency parameter.
//
// An unknown currency is answered with HTTP 400 here - never silently falling
// back to the base currency, which would masquerade Passthrough as Estimate.
// ok is false when the request has already been answered.
//
// Whether a Ratio actually exists is not checked: the services report that by
// returning nothing, which the handlers map to their empty response shape.
func (s *Server) resolveConvertCurrency(w http.ResponseWriter, r *http.Request) (string, bool) {
	currency := getParamLowercase(r, convertCurrencyParam)
	if currency == "" {
		return "", true
	}

	if s.currencyRatiosService == nil || !s.currencyRatiosService.IsCurrencySupported(currency) {
		s.sendJSONError(w, http.StatusBadRequest, fmt.Sprintf("unsupported %s: %s", convertCurrencyParam, currency))
		return "", false
	}

	return currency, true
}

// handleLeaderboardMarkets responds with market data from the leaderboard service
func (s *Server) handleLeaderboardMarkets(w http.ResponseWriter, r *http.Request) {
	convertCurrency, ok := s.resolveConvertCurrency(w, r)
	if !ok {
		return
	}

	data := s.cgService.GetCacheData(convertCurrency)
	if data == nil {
		s.sendJSONResponse(w, emptyMarketsResponse())
		return
	}

	s.sendJSONResponse(w, data)
}

// handleLeaderboardPrices responds with price quotes from Binance service
func (s *Server) handleLeaderboardPrices(w http.ResponseWriter, r *http.Request) {
	s.handleLeaderboardSimplePrices(w, r)
}

// handleLeaderboardSimplePrices responds with simple price quotes filtered by currency
func (s *Server) handleLeaderboardSimplePrices(w http.ResponseWriter, r *http.Request) {
	convertCurrency, ok := s.resolveConvertCurrency(w, r)
	if !ok {
		return
	}

	currency := getParamLowercase(r, "currency")

	prices := s.cgService.GetTopPricesQuotes(currency, convertCurrency)
	if prices == nil {
		prices = interfaces.LeaderboardQuotes{}
	}

	s.sendJSONResponse(w, prices)
}
