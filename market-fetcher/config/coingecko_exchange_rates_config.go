package config

import "time"

// ExchangeRatesFetcherConfig defines configuration for the CoinGecko exchange rates
// Passthrough service (/api/v3/exchange_rates served verbatim).
type ExchangeRatesFetcherConfig struct {
	// UpdateInterval is how often the exchange rates snapshot is refreshed
	UpdateInterval time.Duration `yaml:"update_interval"`
}

// GetDefaultExchangeRatesConfig returns the default exchange rates configuration
func GetDefaultExchangeRatesConfig() ExchangeRatesFetcherConfig {
	return ExchangeRatesFetcherConfig{
		UpdateInterval: 5 * time.Minute,
	}
}
