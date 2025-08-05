package config

import (
	"fmt"
	"sort"
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

// MarketTier defines a tier configuration for token ranges
type MarketTier struct {
	Name           string        `yaml:"name"`            // Name of the tier (e.g., "tier1", "tier2")
	RangeFrom      int           `yaml:"range_from"`      // Start of token range (1-based)
	RangeTo        int           `yaml:"range_to"`        // End of token range (inclusive)
	UpdateInterval time.Duration `yaml:"update_interval"` // Update interval for this tier
	TTL            time.Duration `yaml:"ttl"`             // TTL for this tier's data
}

type CoingeckoMarketsFetcher struct {
	RequestDelay          time.Duration          `yaml:"request_delay"`           // Delay between requests
	MarketParamsNormalize *MarketParamsNormalize `yaml:"market_params_normalize"` // Parameters normalization config
	Currency              string                 `yaml:"currency"`                // Currency for market data
	Tiers                 []MarketTier           `yaml:"tiers"`                   // Tier configurations
	DefaultTTL            time.Duration          `yaml:"default_ttl"`             // Default TTL for non-tier operations
}

// Validate validates the CoingeckoMarketsFetcher configuration
func (c *CoingeckoMarketsFetcher) Validate() error {
	if err := c.validateTiers(); err != nil {
		return fmt.Errorf("tier configuration validation failed: %w", err)
	}

	return nil
}

// validateTiers validates that tier ranges don't overlap and are valid
func (c *CoingeckoMarketsFetcher) validateTiers() error {
	if len(c.Tiers) == 0 {
		return fmt.Errorf("at least one tier must be configured")
	}

	// Create a copy of tiers and sort by RangeFrom for easier validation
	tiers := make([]MarketTier, len(c.Tiers))
	copy(tiers, c.Tiers)
	sort.Slice(tiers, func(i, j int) bool {
		return tiers[i].RangeFrom < tiers[j].RangeFrom
	})

	for i, tier := range tiers {
		// Validate individual tier
		if tier.Name == "" {
			return fmt.Errorf("tier at index %d: name cannot be empty", i)
		}
		if tier.RangeFrom <= 0 {
			return fmt.Errorf("tier '%s': range_from must be greater than 0, got %d", tier.Name, tier.RangeFrom)
		}
		if tier.RangeTo < tier.RangeFrom {
			return fmt.Errorf("tier '%s': range_to (%d) must be >= range_from (%d)", tier.Name, tier.RangeTo, tier.RangeFrom)
		}
		if tier.UpdateInterval <= 0 {
			return fmt.Errorf("tier '%s': update_interval must be greater than 0", tier.Name)
		}
		if tier.TTL <= 0 {
			return fmt.Errorf("tier '%s': ttl must be greater than 0", tier.Name)
		}

		// Check for overlaps with previous tier
		if i > 0 {
			prevTier := tiers[i-1]
			if tier.RangeFrom <= prevTier.RangeTo {
				return fmt.Errorf("tier '%s' range [%d-%d] overlaps with tier '%s' range [%d-%d]",
					tier.Name, tier.RangeFrom, tier.RangeTo,
					prevTier.Name, prevTier.RangeFrom, prevTier.RangeTo)
			}
		}
	}

	return nil
}

func (c *CoingeckoMarketsFetcher) GetTTL() time.Duration {
	if c.DefaultTTL > 0 {
		return c.DefaultTTL
	}

	return 30 * time.Minute
}
