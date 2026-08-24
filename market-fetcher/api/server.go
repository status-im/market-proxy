package api

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/status-im/market-proxy/coingecko_assets_platforms"
	"github.com/status-im/market-proxy/coingecko_coins"
	"github.com/status-im/market-proxy/coingecko_market_chart"
	"github.com/status-im/market-proxy/coingecko_token_list"
	"github.com/status-im/market-proxy/interfaces"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/status-im/market-proxy/coingecko_exchange_rates"
	coingecko "github.com/status-im/market-proxy/coingecko_leaderboard"
	"github.com/status-im/market-proxy/coingecko_tokens"
	"github.com/status-im/market-proxy/currency_ratios"
)

// ILeaderboardService is the part of the leaderboard service the HTTP layer uses
type ILeaderboardService interface {
	GetCacheData() *coingecko.APIResponse
	GetTopPricesQuotes(currency string) map[string]coingecko.Quote
	Healthy() bool
}

// IMarketsHandlerService is the part of the markets service the HTTP layer uses
type IMarketsHandlerService interface {
	interfaces.IMarketsService
	Healthy() bool
}

// IPricesHandlerService is the part of the prices service the HTTP layer uses
type IPricesHandlerService interface {
	interfaces.IPricesService
	Healthy() bool
}

// IExchangeRatesService serves the cached CoinGecko exchange rates Passthrough body
type IExchangeRatesService interface {
	ExchangeRates() (coingecko_exchange_rates.ExchangeRatesResponse, error)
}

type Server struct {
	port                   string
	cgService              ILeaderboardService
	tokensService          *coingecko_tokens.Service
	pricesService          IPricesHandlerService
	marketsService         IMarketsHandlerService
	marketChartService     *coingecko_market_chart.Service
	assetsPlatformsService *coingecko_assets_platforms.Service
	tokenListService       *coingecko_token_list.Service
	coinsService           *coingecko_coins.Service
	currencyRatiosService  currency_ratios.IProvider
	exchangeRatesService   IExchangeRatesService
	server                 *http.Server
}

func New(port string, cgService ILeaderboardService, tokensService *coingecko_tokens.Service, pricesService IPricesHandlerService, marketsService IMarketsHandlerService, marketChartService *coingecko_market_chart.Service, assetsPlatformsService *coingecko_assets_platforms.Service, tokenListService *coingecko_token_list.Service, coinsService *coingecko_coins.Service, currencyRatiosService currency_ratios.IProvider, exchangeRatesService IExchangeRatesService) *Server {
	return &Server{
		port:                   port,
		cgService:              cgService,
		tokensService:          tokensService,
		pricesService:          pricesService,
		marketsService:         marketsService,
		marketChartService:     marketChartService,
		assetsPlatformsService: assetsPlatformsService,
		tokenListService:       tokenListService,
		coinsService:           coinsService,
		currencyRatiosService:  currencyRatiosService,
		exchangeRatesService:   exchangeRatesService,
	}
}

func (s *Server) Start(ctx context.Context) error {
	router := mux.NewRouter()

	// Existing endpoints
	router.HandleFunc("/api/v1/leaderboard/prices", s.handleLeaderboardPrices)
	router.HandleFunc("/api/v1/leaderboard/simpleprices", s.handleLeaderboardSimplePrices)
	router.HandleFunc("/api/v1/leaderboard/markets", s.handleLeaderboardMarkets)
	router.HandleFunc("/api/v1/asset_platforms", s.handleAssetsPlatforms)
	router.HandleFunc("/api/v1/exchange_rates", s.handleExchangeRates)
	router.HandleFunc("/api/v1/simple/price", s.handleSimplePrice)

	// All coins endpoints are handled by the coins router
	router.PathPrefix("/api/v1/coins/").HandlerFunc(s.handleCoinsRoutes)

	// Token list endpoint
	router.HandleFunc("/api/v1/token_lists/{platform}/all.json", s.TokenListHandler).Methods("GET")

	router.HandleFunc("/health", s.handleHealth)
	router.Handle("/metrics", promhttp.Handler())

	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: router,
	}

	log.Printf("Server starting at http://localhost:%s", s.port)
	log.Println("Prometheus metrics available at /metrics endpoint")

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	return nil
}
