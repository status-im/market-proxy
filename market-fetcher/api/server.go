package api

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/status-im/market-proxy/binance"
	coingecko "github.com/status-im/market-proxy/coingecko_leaderboard"
	"github.com/status-im/market-proxy/tokens"
)

type Server struct {
	port           string
	binanceService *binance.Service
	cgService      *coingecko.Service
	tokensService  *tokens.Service
	server         *http.Server
}

func New(port string, binanceService *binance.Service, cgService *coingecko.Service, tokensService *tokens.Service) *Server {
	return &Server{
		port:           port,
		binanceService: binanceService,
		cgService:      cgService,
		tokensService:  tokensService,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/leaderboard/prices", s.handleLeaderboardPrices)
	mux.HandleFunc("/api/v1/leaderboard/markets", s.handleLeaderboardMarkets)
	mux.HandleFunc("/api/v1/coins/list", s.handleCoinsList)
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}

	log.Printf("Server starting at http://localhost:%s", s.port)
	log.Println("Prometheus metrics available at /metrics endpoint")

	// Start the server in a goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	return nil
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

// handleCoinsList responds with the list of tokens filtered by supported platforms
// This endpoint mimics the CoinGecko API endpoint /api/v3/coins/list with platform information
func (s *Server) handleCoinsList(w http.ResponseWriter, r *http.Request) {
	// Check for include_platform parameter, though we always include platforms
	includePlatform := r.URL.Query().Get("include_platform")
	if includePlatform != "" {
		include, err := strconv.ParseBool(includePlatform)
		if err != nil || !include {
			http.Error(w, "include_platform parameter must be a valid boolean value representing 'true'", http.StatusBadRequest)
			return
		}
	}

	tokens := s.tokensService.GetTokens()
	if len(tokens) == 0 {
		http.Error(w, "No token data available", http.StatusServiceUnavailable)
		return
	}

	s.sendJSONResponse(w, tokens)
}

// handleHealth responds with 200 OK to indicate the service is running
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check if services are initialized
	status := map[string]interface{}{
		"status": "ok",
		"services": map[string]string{
			"binance":   "unknown",
			"coingecko": "unknown",
			"tokens":    "unknown",
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

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}
}
