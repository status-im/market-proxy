package config

import (
	"time"
)

// MarketParamsNormalize defines configuration for normalizing market parameters
type MarketParamsNormalize struct {
	VsCurrency            *string `yaml:"vs_currency,omitempty"`
	Order                 *string `yaml:"order,omitempty"`
	PerPage               *int    `yaml:"per_page,omitempty"`
	Sparkline             *bool   `yaml:"sparkline,omitempty"`
	PriceChangePercentage *string `yaml:"price_change_percentage,omitempty"`
	Category              *string `yaml:"category,omitempty"`
}

type CoingeckoMarketsFetcher struct {
	RequestDelay             time.Duration          `yaml:"request_delay"`               // Delay between requests
	TTL                      time.Duration          `yaml:"ttl"`                         // Time to live for cached market data
	MarketParamsNormalize    *MarketParamsNormalize `yaml:"market_params_normalize"`     // Parameters normalization config
	TopMarketsUpdateInterval time.Duration          `yaml:"top_markets_update_interval"` // Interval for periodic top markets updates
	TopMarketsLimit          int                    `yaml:"top_markets_limit"`           // Limit for top markets to fetch
	Currency                 string                 `yaml:"currency"`                    // Currency for market data
}
