package api

import (
	"net/http"
)

// handleHealth responds with 200 OK to indicate the service is running
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check if services are initialized
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
		},
	}

	// Check if Binance service is healthy (has received at least one update)
	if s.binanceService.Healthy() {
		status["services"].(map[string]string)["binance"] = "up"
	}

	// Check if CoinGecko service is healthy (can fetch at least one page)
	if s.cgService.Healthy() {
		status["services"].(map[string]string)["coingecko"] = "up"
	}

	// Check if Tokens service is healthy
	if s.tokensService.Healthy() {
		status["services"].(map[string]string)["tokens"] = "up"
	}

	// Check if CoinGecko Prices service is healthy
	if s.pricesService.Healthy() {
		status["services"].(map[string]string)["coingecko_prices"] = "up"
	}

	// Check if CoinGecko Markets service is healthy
	if s.marketsService.Healthy() {
		status["services"].(map[string]string)["coingecko_markets"] = "up"
	}

	// Check if CoinGecko Market Chart service is healthy
	if s.marketChartService.Healthy() {
		status["services"].(map[string]string)["coingecko_market_chart"] = "up"
	}

	// Check if CoinGecko Assets Platforms service is healthy
	if s.assetsPlatformsService.Healthy() {
		status["services"].(map[string]string)["coingecko_platforms"] = "up"
	}

	s.sendJSONResponse(w, status)
}
