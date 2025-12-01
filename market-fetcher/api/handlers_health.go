package api

import (
	"net/http"
)

// handleHealth responds with 200 OK to indicate the service is running
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "ok",
		"services": map[string]string{
			"binance":                "unknown",
			"coingecko":              "unknown",
			"tokens":                 "unknown",
			"coingecko_prices":       "unknown",
			"coingecko_markets":      "unknown",
			"coingecko_market_chart": "unknown",
			"coingecko_platforms":    "unknown",
			"coingecko_coins":        "unknown",
		},
	}

	if s.binanceService.Healthy() {
		status["services"].(map[string]string)["binance"] = "up"
	}

	if s.cgService.Healthy() {
		status["services"].(map[string]string)["coingecko"] = "up"
	}

	if s.tokensService.Healthy() {
		status["services"].(map[string]string)["tokens"] = "up"
	}

	if s.pricesService.Healthy() {
		status["services"].(map[string]string)["coingecko_prices"] = "up"
	}

	if s.marketsService.Healthy() {
		status["services"].(map[string]string)["coingecko_markets"] = "up"
	}

	if s.marketChartService.Healthy() {
		status["services"].(map[string]string)["coingecko_market_chart"] = "up"
	}

	if s.assetsPlatformsService.Healthy() {
		status["services"].(map[string]string)["coingecko_platforms"] = "up"
	}

	if s.coinsService.Healthy() {
		status["services"].(map[string]string)["coingecko_coins"] = "up"
	}

	s.sendJSONResponse(w, status)
}
