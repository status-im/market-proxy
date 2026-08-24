package config

import (
	"fmt"
	"time"
)

// CurrencyRatiosConfig defines configuration for the currency ratios service.
//
// The service fetches a single CoinGecko simple/price call for the reference coins
// in every configured currency and derives, per currency, the spot and 24h-ago
// ratio against the base currency (usd). Those ratios are used to compute
// realtime Estimates from cached USD Passthrough values.
type CurrencyRatiosConfig struct {
	// UpdateInterval is how often the reference prices are refreshed
	UpdateInterval time.Duration `yaml:"update_interval"`

	// ReferenceCoins are the coin ids used to derive ratios, in priority order.
	// The reference coin's own price cancels out in the division, so the choice
	// does not affect the resulting ratios - the list only provides a fallback
	// when the first coin's row is missing or incomplete.
	ReferenceCoins []string `yaml:"reference_coins"`

	// Currencies is the list of currencies ratios are computed for.
	// It doubles as the allow-list for the convert_currency query parameter.
	Currencies []string `yaml:"currencies"`
}

// CurrencyRatiosBaseCurrency is the currency all ratios are relative to.
// Leaderboard Passthrough data is cached in this currency.
const CurrencyRatiosBaseCurrency = "usd"

// GetDefaultCurrencyRatiosConfig returns the default currency ratios configuration
func GetDefaultCurrencyRatiosConfig() CurrencyRatiosConfig {
	return CurrencyRatiosConfig{
		UpdateInterval: time.Minute,
		ReferenceCoins: []string{"bitcoin", "ethereum"},
		Currencies:     []string{CurrencyRatiosBaseCurrency, "eur", "btc", "eth"},
	}
}

// Validate validates the currency ratios configuration
func (c *CurrencyRatiosConfig) Validate() error {
	if c.UpdateInterval <= 0 {
		return fmt.Errorf("update_interval must be greater than 0")
	}
	if len(c.ReferenceCoins) == 0 {
		return fmt.Errorf("at least one reference coin must be configured")
	}
	if len(c.Currencies) == 0 {
		return fmt.Errorf("at least one currency must be configured")
	}

	hasBase := false
	for _, currency := range c.Currencies {
		if currency == CurrencyRatiosBaseCurrency {
			hasBase = true
			break
		}
	}
	if !hasBase {
		return fmt.Errorf("currencies must contain the base currency %q", CurrencyRatiosBaseCurrency)
	}

	return nil
}

// IsEmpty reports whether no currency ratios configuration was provided
func (c *CurrencyRatiosConfig) IsEmpty() bool {
	return c.UpdateInterval == 0 && len(c.ReferenceCoins) == 0 && len(c.Currencies) == 0
}
