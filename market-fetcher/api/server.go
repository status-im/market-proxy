package api

import (
	"context"
	"log"
	"net/http"

	"github.com/status-im/market-proxy/coingecko_assets_platforms"
	"github.com/status-im/market-proxy/coingecko_markets"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/status-im/market-proxy/binance"
	coingecko "github.com/status-im/market-proxy/coingecko_leaderboard"
	"github.com/status-im/market-proxy/coingecko_prices"
	"github.com/status-im/market-proxy/coingecko_tokens"
)

type Server struct {
	port                   string
	binanceService         *binance.Service
	cgService              *coingecko.Service
	tokensService          *coingecko_tokens.Service
	pricesService          *coingecko_prices.Service
	marketsService         *coingecko_markets.Service
	assetsPlatformsService *coingecko_assets_platforms.Service
	server                 *http.Server
}

func New(port string, binanceService *binance.Service, cgService *coingecko.Service, tokensService *coingecko_tokens.Service, pricesService *coingecko_prices.Service, marketsService *coingecko_markets.Service, assetsPlatformsService *coingecko_assets_platforms.Service) *Server {
	return &Server{
		port:                   port,
		binanceService:         binanceService,
		cgService:              cgService,
		tokensService:          tokensService,
		pricesService:          pricesService,
		marketsService:         marketsService,
		assetsPlatformsService: assetsPlatformsService,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/leaderboard/prices", s.handleLeaderboardPrices)
	mux.HandleFunc("/api/v1/leaderboard/simpleprices", s.handleLeaderboardSimplePrices)
	mux.HandleFunc("/api/v1/leaderboard/markets", s.handleLeaderboardMarkets)
	mux.HandleFunc("/api/v1/coins/list", s.handleCoinsList)
	mux.HandleFunc("/api/v1/coins/markets", s.handleCoinsMarkets)
	mux.HandleFunc("/api/v1/asset_platforms", s.handleAssetsPlatforms)
	mux.HandleFunc("/api/v1/simple/price", s.handleSimplePrice)
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
