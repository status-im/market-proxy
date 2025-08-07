package config

import (
	"time"
)

// MarketChartFetcherConfig defines configuration for CoinGecko market chart service
type MarketChartFetcherConfig struct {
	// HourlyTTL is the cache TTL for hourly data (for requests with days <= DailyDataThreshold)
	HourlyTTL time.Duration `yaml:"hourly_ttl"`

	// DailyTTL is the cache TTL for daily data (for requests with days > DailyDataThreshold)
	DailyTTL time.Duration `yaml:"daily_ttl"`

	// DailyDataThreshold is the number of days after which CoinGecko returns daily data instead of hourly
	// According to CoinGecko API documentation:
	// - 1 day = 5-minutely data
	// - 2-90 days = hourly data
	// - above 90 days = daily data
	DailyDataThreshold int `yaml:"daily_data_threshold"`

	// TryFreeApiFirst determines whether to try free API (no key) first when no interval is specified
	TryFreeApiFirst bool `yaml:"try_free_api_first"`
}

// GetDefaultMarketChartConfig returns default configuration for market chart service
func GetDefaultMarketChartConfig() MarketChartFetcherConfig {
	return MarketChartFetcherConfig{
		HourlyTTL:          30 * time.Minute, // 30 minutes for hourly data
		DailyTTL:           12 * time.Hour,   // 12 hours for daily data
		DailyDataThreshold: 90,               // 90 days is the threshold for daily data
		TryFreeApiFirst:    false,            // Default to false
	}
}
