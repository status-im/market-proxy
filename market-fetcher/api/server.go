package api

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/status-im/market-proxy/binance"
	"github.com/status-im/market-proxy/coingecko"
)

type Server struct {
	port           string
	binanceService *binance.Service
	cgService      *coingecko.Service
}

func New(port string, binanceService *binance.Service, cgService *coingecko.Service) *Server {
	return &Server{
		port:           port,
		binanceService: binanceService,
		cgService:      cgService,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/api/v1/leaderboard/prices", s.handleLeaderboardPrices)
	http.HandleFunc("/api/v1/leaderboard/markets", s.handleLeaderboardMarkets)
	http.HandleFunc("/health", s.handleHealth)

	return http.ListenAndServe(":"+s.port, nil)
}

func (s *Server) handleLeaderboardMarkets(w http.ResponseWriter, r *http.Request) {
	data := s.cgService.GetCacheData()
	if data == nil {
		http.Error(w, "No data available", http.StatusServiceUnavailable)
		return
	}

	s.sendJSONResponse(w, data)
}

func (s *Server) handleLeaderboardPrices(w http.ResponseWriter, r *http.Request) {
	quotes := s.binanceService.GetLatestQuotes()
	if len(quotes) == 0 {
		http.Error(w, "No CoinGecko prices available", http.StatusServiceUnavailable)
		return
	}

	s.sendJSONResponse(w, quotes)
}

// handleHealth responds with 200 OK to indicate the service is running
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check if services are initialized
	status := map[string]interface{}{
		"status": "ok",
		"services": map[string]string{
			"binance":   "unknown",
			"coingecko": "unknown",
		},
	}

	// Check if CoinMarketCap data is available
	if s.binanceService.GetLatestQuotes() != nil {
		status["services"].(map[string]string)["binance"] = "up"
	}

	// Check if CoinGecko data is available
	if s.cgService.GetCacheData() != nil {
		status["services"].(map[string]string)["coingecko"] = "up"
	}

	s.sendJSONResponse(w, status)
}

// sendJSONResponse is a common wrapper for JSON responses that sets Content-Type,
// Content-Length, and ETag headers
func (s *Server) sendJSONResponse(w http.ResponseWriter, data interface{}) {
	// Marshal the data to calculate content length and ETag
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	err := encoder.Encode(data)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}

	responseBytes := buffer.Bytes()

	// Calculate ETag (MD5 hash of the response)
	hash := md5.Sum(responseBytes)
	etag := hex.EncodeToString(hash[:])

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(responseBytes)))
	w.Header().Set("ETag", "\""+etag+"\"")

	// Write the response
	w.Write(responseBytes)
}
