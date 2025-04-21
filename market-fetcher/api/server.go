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
	"github.com/status-im/market-proxy/coingecko"
)

type Server struct {
	port           string
	binanceService *binance.Service
	cgService      *coingecko.Service
	server         *http.Server
}

func New(port string, binanceService *binance.Service, cgService *coingecko.Service) *Server {
	return &Server{
		port:           port,
		binanceService: binanceService,
		cgService:      cgService,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/leaderboard/prices", s.handleLeaderboardPrices)
	mux.HandleFunc("/api/v1/leaderboard/markets", s.handleLeaderboardMarkets)
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}

	log.Printf("Server started at http://localhost:%s", s.port)
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

	// Check if Binance service is healthy (has received at least one update)
	if s.binanceService.Healthy() {
		status["services"].(map[string]string)["binance"] = "up"
	}

	// Check if CoinGecko service is healthy (can fetch at least one page)
	if s.cgService.Healthy() {
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

func (s *Server) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}
