package core

import (
	"context"
	"os"
	"strings"

	"github.com/status-im/market-proxy/api"
	"github.com/status-im/market-proxy/binance"
	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/coingecko_assets_platforms"
	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/coingecko_leaderboard"
	"github.com/status-im/market-proxy/coingecko_market_chart"
	"github.com/status-im/market-proxy/coingecko_markets"
	"github.com/status-im/market-proxy/coingecko_prices"
	"github.com/status-im/market-proxy/coingecko_token_list"
	"github.com/status-im/market-proxy/coingecko_tokens"
	"github.com/status-im/market-proxy/config"
)

// Setup creates and registers all services
func Setup(ctx context.Context, cfg *config.Config) (*Registry, error) {
	registry := NewRegistry()

	// Apply API key rate limiter settings
	cg.GetRateLimiterManagerInstance().SetConfig(cfg.APIKeySettings)

	// ICache service
	cacheService := cache.NewService(cfg.Cache)
	registry.Register(cacheService)

	// Tokens service
	tokensService := coingecko_tokens.NewService(cfg)
	registry.Register(tokensService)

	// Token List service
	tokenListService := coingecko_token_list.NewService(cfg)
	registry.Register(tokenListService)

	// Markets service
	marketsService := coingecko_markets.NewService(cacheService, cfg, tokensService)
	registry.Register(marketsService)

	// Prices service
	pricesService := coingecko_prices.NewService(cacheService, cfg, marketsService, tokensService)
	registry.Register(pricesService)

	// MarketChart service
	marketChartService := coingecko_market_chart.NewService(cacheService, cfg)
	registry.Register(marketChartService)

	// Assets Platforms service
	assetsPlatformsService := coingecko_assets_platforms.NewService(cfg)
	registry.Register(assetsPlatformsService)

	// Binance service
	// TODO: remove #43
	binanceService := binance.NewService(cfg)
	registry.Register(binanceService)

	// Leaderboard service
	cgService := coingecko_leaderboard.NewService(cfg, pricesService, marketsService)
	registry.Register(cgService)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// HTTP Server
	server := api.New(port, binanceService, cgService, tokensService, pricesService, marketsService, marketChartService, assetsPlatformsService, tokenListService)
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
	select {
	case <-ctx.Done():
		return // Context is cancelled, do nothing
	default:
	}

	cgData := cgService.GetCacheData()
	if cgData != nil {
		symbols := make([]string, 0, len(cgData.Data))
		for _, coin := range cgData.Data {
			symbols = append(symbols, strings.ToUpper(coin.Symbol))
		}

		binanceService.SetWatchList(symbols, "USDT")
	}
}
