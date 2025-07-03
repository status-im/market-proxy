package api

import (
	"net/http"
)

// handleLeaderboardMarkets responds with market data from the leaderboard service
func (s *Server) handleLeaderboardMarkets(w http.ResponseWriter, r *http.Request) {
	data := s.cgService.GetCacheData()
	if data == nil {
		http.Error(w, "No data available", http.StatusServiceUnavailable)
		return
	}

	s.sendJSONResponse(w, data)
}

// handleLeaderboardPrices responds with price quotes from Binance service
func (s *Server) handleLeaderboardPrices(w http.ResponseWriter, r *http.Request) {
	quotes := s.binanceService.GetLatestQuotes()
	if len(quotes) == 0 {
		http.Error(w, "No CoinGecko prices available", http.StatusServiceUnavailable)
		return
	}

	s.sendJSONResponse(w, quotes)
}

// handleLeaderboardSimplePrices responds with simple price quotes filtered by currency
func (s *Server) handleLeaderboardSimplePrices(w http.ResponseWriter, r *http.Request) {
	// Parse currency parameter
	currency := r.URL.Query().Get("currency")

	prices := s.cgService.GetTopPricesQuotes(currency)
	s.sendJSONResponse(w, prices)
}
