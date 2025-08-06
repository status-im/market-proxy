package config

import (
	"fmt"
	"sort"
	"time"
)

// PriceTier defines a tier configuration for token ranges
type PriceTier struct {
	Name              string        `yaml:"name"`                // Name of the tier (e.g., "top-1000", "top-1001-10000")
	TokenFrom         int           `yaml:"token_from"`          // Start of token range (1-based)
	TokenTo           int           `yaml:"token_to"`            // End of token range (inclusive)
	UpdateInterval    time.Duration `yaml:"update_interval"`     // Update interval for this tier
	FetchCoinslistIds bool          `yaml:"fetch_coinslist_ids"` // Whether to fetch missing coinslist IDs for supported platforms after main fetch
}

// CoingeckoPricesFetcher represents configuration for CoinGecko prices service
type CoingeckoPricesFetcher struct {
	ChunkSize    int           `yaml:"chunk_size"`    // Number of tokens to fetch in one request
	RequestDelay time.Duration `yaml:"request_delay"` // Delay between requests
	Currencies   []string      `yaml:"currencies"`    // Default currencies to fetch
	TTL          time.Duration `yaml:"ttl"`           // Time to live for cached price data
	Tiers        []PriceTier   `yaml:"tiers"`         // Tier configurations
}

// Validate validates the CoingeckoPricesFetcher configuration
func (c *CoingeckoPricesFetcher) Validate() error {
	if err := c.validateTiers(); err != nil {
		return fmt.Errorf("tier configuration validation failed: %w", err)
	}

	return nil
}

// validateTiers validates that tier ranges don't overlap and are valid
func (c *CoingeckoPricesFetcher) validateTiers() error {
	if len(c.Tiers) == 0 {
		return fmt.Errorf("at least one tier must be configured")
	}

	// Create a copy of tiers and sort by TokenFrom for easier validation
	tiers := make([]PriceTier, len(c.Tiers))
	copy(tiers, c.Tiers)
	sort.Slice(tiers, func(i, j int) bool {
		return tiers[i].TokenFrom < tiers[j].TokenFrom
	})

	for i, tier := range tiers {
		// Validate individual tier
		if tier.Name == "" {
			return fmt.Errorf("tier at index %d: name cannot be empty", i)
		}
		if tier.TokenFrom <= 0 {
			return fmt.Errorf("tier '%s': token_from must be greater than 0, got %d", tier.Name, tier.TokenFrom)
		}
		if tier.TokenTo < tier.TokenFrom {
			return fmt.Errorf("tier '%s': token_to (%d) must be >= token_from (%d)", tier.Name, tier.TokenTo, tier.TokenFrom)
		}
		if tier.UpdateInterval <= 0 {
			return fmt.Errorf("tier '%s': update_interval must be greater than 0", tier.Name)
		}

		// Check for overlaps with previous tier
		if i > 0 {
			prevTier := tiers[i-1]
			if tier.TokenFrom <= prevTier.TokenTo {
				return fmt.Errorf("tier '%s' token [%d-%d] overlaps with tier '%s' token [%d-%d]",
					tier.Name, tier.TokenFrom, tier.TokenTo,
					prevTier.Name, prevTier.TokenFrom, prevTier.TokenTo)
			}
		}
	}

	return nil
}

// GetTTL returns the TTL configuration or default value
func (c *CoingeckoPricesFetcher) GetTTL() time.Duration {
	if c.TTL > 0 {
		return c.TTL
	}

	return 30 * time.Second
}
