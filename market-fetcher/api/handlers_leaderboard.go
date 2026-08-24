package api

import (
	"fmt"
	"net/http"

	"github.com/status-im/market-proxy/coingecko_leaderboard"
	"github.com/status-im/market-proxy/currency_ratios"
)

// convertCurrencyParam is the optional query parameter requesting a realtime
// Estimate in another currency. It is deliberately separate from `currency`,
// which keeps its Passthrough-selection semantics.
const convertCurrencyParam = "convert_currency"

// convertOutcome describes what resolveConvertRatio decided about a request
type convertOutcome int

const (
	// convertNone - no conversion was requested, serve Passthrough as before
	convertNone convertOutcome = iota
	// convertReady - conversion was requested and a ratio is available
	convertReady
	// convertUnavailable - conversion was requested for a supported currency,
	// but no ratio exists yet; serve the endpoint's empty response shape
	convertUnavailable
	// convertRejected - the request was already answered with an error
	convertRejected
)

// emptyMarketsResponse is the response shape used when no markets data is available
func emptyMarketsResponse() map[string]interface{} {
	return map[string]interface{}{
		"data": []interface{}{},
	}
}

// resolveConvertRatio resolves the convert_currency parameter into a ratio.
//
// An unknown currency is answered with HTTP 400 here - never silently falling
// back to the base currency, which would masquerade Passthrough as Estimate.
func (s *Server) resolveConvertRatio(w http.ResponseWriter, r *http.Request) (string, currency_ratios.Ratio, convertOutcome) {
	currency := getParamLowercase(r, convertCurrencyParam)
	if currency == "" {
		return "", currency_ratios.Ratio{}, convertNone
	}

	if s.currencyRatiosService == nil || !s.currencyRatiosService.IsCurrencySupported(currency) {
		s.sendJSONError(w, http.StatusBadRequest, fmt.Sprintf("unsupported %s: %s", convertCurrencyParam, currency))
		return currency, currency_ratios.Ratio{}, convertRejected
	}

	// Supported currency but no ratio yet (service just started, or the provider
	// skipped this currency in the latest snapshot): behave like an empty cache.
	ratio, found := s.currencyRatiosService.GetSnapshot().Ratio(currency)
	if !found {
		return currency, currency_ratios.Ratio{}, convertUnavailable
	}

	return currency, ratio, convertReady
}

// handleLeaderboardMarkets responds with market data from the leaderboard service
func (s *Server) handleLeaderboardMarkets(w http.ResponseWriter, r *http.Request) {
	_, ratio, outcome := s.resolveConvertRatio(w, r)

	switch outcome {
	case convertRejected:
		return
	case convertUnavailable:
		s.sendJSONResponse(w, emptyMarketsResponse())
		return
	}

	data := s.cgService.GetCacheData()
	if data == nil {
		s.sendJSONResponse(w, emptyMarketsResponse())
		return
	}

	if outcome == convertReady {
		data = coingecko_leaderboard.ConvertAPIResponse(data, ratio)
	}

	s.sendJSONResponse(w, data)
}

// handleLeaderboardPrices responds with price quotes from Binance service
func (s *Server) handleLeaderboardPrices(w http.ResponseWriter, r *http.Request) {
	s.handleLeaderboardSimplePrices(w, r)
}

// handleLeaderboardSimplePrices responds with simple price quotes filtered by currency
func (s *Server) handleLeaderboardSimplePrices(w http.ResponseWriter, r *http.Request) {
	_, ratio, outcome := s.resolveConvertRatio(w, r)

	switch outcome {
	case convertRejected:
		return
	case convertUnavailable:
		s.sendJSONResponse(w, coingecko_leaderboard.PriceQuotes{})
		return
	case convertReady:
		// Estimates are always computed from the base currency Passthrough data
		prices := s.cgService.GetTopPricesQuotes(currency_ratios.BaseCurrency)
		s.sendJSONResponse(w, coingecko_leaderboard.ConvertQuotes(prices, ratio))
		return
	}

	// Parse currency parameter
	currency := getParamLowercase(r, "currency")

	prices := s.cgService.GetTopPricesQuotes(currency)
	s.sendJSONResponse(w, prices)
}
