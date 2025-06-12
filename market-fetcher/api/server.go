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
	"strings"
	"time"

	"github.com/status-im/market-proxy/coingecko_common"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/status-im/market-proxy/binance"
	coingecko "github.com/status-im/market-proxy/coingecko_leaderboard"
	"github.com/status-im/market-proxy/coingecko_prices"
	"github.com/status-im/market-proxy/coingecko_tokens"
)

type Server struct {
	port           string
	binanceService *binance.Service
	cgService      *coingecko.Service
	tokensService  *coingecko_tokens.Service
	pricesService  *coingecko_prices.Service
	server         *http.Server
}

func New(port string, binanceService *binance.Service, cgService *coingecko.Service, tokensService *coingecko_tokens.Service, pricesService *coingecko_prices.Service) *Server {
	return &Server{
		port:           port,
		binanceService: binanceService,
		cgService:      cgService,
		tokensService:  tokensService,
		pricesService:  pricesService,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/leaderboard/prices", s.handleLeaderboardPrices)
	mux.HandleFunc("/api/v1/leaderboard/simpleprices", s.handleLeaderboardSimplePrices)
	mux.HandleFunc("/api/v1/leaderboard/markets", s.handleLeaderboardMarkets)
	mux.HandleFunc("/api/v1/coins/list", s.handleCoinsList)
	mux.HandleFunc("/api/v1/simple/price", s.handleSimplePrice)
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}

	log.Printf("Server starting at http://localhost:%s", s.port)
	log.Println("Prometheus metrics available at /metrics endpoint")

	// Create error channel for server
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for either context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errChan:
		return err
	}
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
func (s *Server) handleLeaderboardSimplePrices(w http.ResponseWriter, r *http.Request) {
	// Parse currency parameter
	currency := r.URL.Query().Get("currency")

	prices := s.cgService.GetTopPricesQuotes(currency)
	s.sendJSONResponse(w, prices)
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
			"binance":          "unknown",
			"coingecko":        "unknown",
			"tokens":           "unknown",
			"coingecko_prices": "unknown",
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

	s.sendJSONResponse(w, status)
}

// handleSimplePrice implements CoinGecko-compatible /api/v3/simple/price endpoint
func (s *Server) handleSimplePrice(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	params := coingecko_common.PriceParams{}

	// Parse IDs (required)
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		http.Error(w, "Parameter 'ids' is required", http.StatusBadRequest)
		return
	}
	params.IDs = strings.Split(idsParam, ",")

	// Parse currencies (vs_currencies, required)
	currenciesParam := r.URL.Query().Get("vs_currencies")
	if currenciesParam == "" {
		http.Error(w, "Parameter 'vs_currencies' is required", http.StatusBadRequest)
		return
	}
	params.Currencies = strings.Split(currenciesParam, ",")

	// Parse optional boolean parameters
	if marketCapParam := r.URL.Query().Get("include_market_cap"); marketCapParam != "" {
		if marketCap, err := strconv.ParseBool(marketCapParam); err == nil {
			params.IncludeMarketCap = marketCap
		}
	}

	if volParam := r.URL.Query().Get("include_24hr_vol"); volParam != "" {
		if vol, err := strconv.ParseBool(volParam); err == nil {
			params.Include24hrVol = vol
		}
	}

	if changeParam := r.URL.Query().Get("include_24hr_change"); changeParam != "" {
		if change, err := strconv.ParseBool(changeParam); err == nil {
			params.Include24hrChange = change
		}
	}

	if updatedParam := r.URL.Query().Get("include_last_updated_at"); updatedParam != "" {
		if updated, err := strconv.ParseBool(updatedParam); err == nil {
			params.IncludeLastUpdatedAt = updated
		}
	}

	// Call price service
	response, err := s.pricesService.SimplePrices(params)
	if err != nil {
		http.Error(w, "Failed to fetch prices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send raw JSON response from cache (CoinGecko format)
	s.sendJSONResponse(w, response)
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
	if _, err := w.Write(responseBytes); err != nil {
		log.Printf("Error writing response: %v", err)
		return
	}
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
