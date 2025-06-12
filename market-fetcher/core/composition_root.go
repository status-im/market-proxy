package core

import (
	"context"
	"os"
	"strings"

	"github.com/status-im/market-proxy/api"
	"github.com/status-im/market-proxy/binance"
	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/coingecko_leaderboard"
	"github.com/status-im/market-proxy/coingecko_prices"
	"github.com/status-im/market-proxy/coingecko_tokens"
	"github.com/status-im/market-proxy/config"
)

// Setup creates and registers all services
func Setup(ctx context.Context, cfg *config.Config) (*Registry, error) {
	registry := NewRegistry()

	// Create Cache service
	cacheService := cache.NewService(cfg.Cache)
	registry.Register(cacheService)

	// Create CoinGecko Prices service with cache dependency
	pricesService := coingecko_prices.NewService(cacheService, cfg)
	registry.Register(pricesService)

	// Create Binance core
	binanceService := binance.NewService(cfg)
	registry.Register(binanceService)

	// Create CoinGecko core with callback and price fetcher
	cgService := coingecko_leaderboard.NewService(cfg, pricesService)
	registry.Register(cgService)

	// Create Tokens core
	tokensService := coingecko_tokens.NewService(cfg)
	registry.Register(tokensService)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server and register it as a core
	server := api.New(port, binanceService, cgService, tokensService, pricesService)
	registry.Register(server)

	// Set update callback directly to our watchlist update function
	cgService.SetOnUpdateCallback(func() {
		// Create a new closure that captures the required variables
		go updateBinanceWatchlist(ctx, cgService, binanceService)
	})
	return registry, nil
}

// updateBinanceWatchlist updates Binance watchlist with data from CoinGecko
func updateBinanceWatchlist(ctx context.Context, cgService *coingecko_leaderboard.Service, binanceService *binance.Service) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return // Context is cancelled, do nothing
	default:
		// Continue processing
	}

	// Get latest data from CoinGecko cache
	cgData := cgService.GetCacheData()
	if cgData != nil {
		// Extract symbols
		symbols := make([]string, 0, len(cgData.Data))
		for _, coin := range cgData.Data {
			// Convert symbols to uppercase as Binance API requires
			symbols = append(symbols, strings.ToUpper(coin.Symbol))
		}

		// Update Binance watchlist with a specific name
		binanceService.SetWatchList(symbols, "USDT")
	}
}
