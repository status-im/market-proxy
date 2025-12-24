package core

import (
	"context"
	"os"

	"github.com/status-im/market-proxy/api"
	"github.com/status-im/market-proxy/cache"
	"github.com/status-im/market-proxy/coingecko_assets_platforms"
	"github.com/status-im/market-proxy/coingecko_coins"
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

	// Coins service
	coinsService := coingecko_coins.NewService(cfg, marketsService, cacheService)
	registry.Register(coinsService)

	// Prices service
	pricesService := coingecko_prices.NewService(cacheService, cfg, marketsService, tokensService)
	registry.Register(pricesService)

	// MarketChart service
	marketChartService := coingecko_market_chart.NewService(cacheService, cfg)
	registry.Register(marketChartService)

	// Assets Platforms service
	assetsPlatformsService := coingecko_assets_platforms.NewService(cfg)
	registry.Register(assetsPlatformsService)

	// Leaderboard service
	cgService := coingecko_leaderboard.NewService(cfg, pricesService, marketsService)
	registry.Register(cgService)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// HTTP Server
	server := api.New(port, cgService, tokensService, pricesService, marketsService, marketChartService, assetsPlatformsService, tokenListService, coinsService)
	registry.Register(server)

	return registry, nil
}
