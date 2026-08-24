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
	IncludeRehypothecated *bool   `yaml:"include_rehypothecated,omitempty"`
}

// MarketTier defines a tier configuration for token pages
type MarketTier struct {
	Name              string        `yaml:"name"`                // Name of the tier (e.g., "tier1", "tier2")
	PageFrom          int           `yaml:"page_from"`           // Start of token page (1-based)
	PageTo            int           `yaml:"page_to"`             // End of token page (inclusive)
	UpdateInterval    time.Duration `yaml:"update_interval"`     // Update interval for this tier
	FetchCoinslistIds bool          `yaml:"fetch_coinslist_ids"` // Whether to fetch missing coinslist IDs for supported platforms after main fetch
}

type MarketsFetcherConfig struct {
	RequestDelay          time.Duration          `yaml:"request_delay"`           // Delay between requests
	MarketParamsNormalize *MarketParamsNormalize `yaml:"market_params_normalize"` // Parameters normalization config
	Tiers                 []MarketTier           `yaml:"tiers"`                   // Tier configurations
	TTL                   time.Duration          `yaml:"ttl"`                     // Default TTL for non-tier operations
}

// Validate validates the MarketsFetcherConfig configuration
func (c *MarketsFetcherConfig) Validate() error {
	if err := c.validateTiers(); err != nil {
		return fmt.Errorf("tier configuration validation failed: %w", err)
	}

	return nil
}

// validateTiers validates that tier ranges don't overlap and are valid
func (c *MarketsFetcherConfig) validateTiers() error {
	if len(c.Tiers) == 0 {
		return fmt.Errorf("at least one tier must be configured")
	}

	// Create a copy of tiers and sort by PageFrom for easier validation
	tiers := make([]MarketTier, len(c.Tiers))
	copy(tiers, c.Tiers)
	sort.Slice(tiers, func(i, j int) bool {
		return tiers[i].PageFrom < tiers[j].PageFrom
	})

	for i, tier := range tiers {
		// Validate individual tier
		if tier.Name == "" {
			return fmt.Errorf("tier at index %d: name cannot be empty", i)
		}
		if tier.PageFrom <= 0 {
			return fmt.Errorf("tier '%s': page_from must be greater than 0, got %d", tier.Name, tier.PageFrom)
		}
		if tier.PageTo < tier.PageFrom {
			return fmt.Errorf("tier '%s': page_to (%d) must be >= page_from (%d)", tier.Name, tier.PageTo, tier.PageFrom)
		}
		if tier.UpdateInterval <= 0 {
			return fmt.Errorf("tier '%s': update_interval must be greater than 0", tier.Name)
		}

		// Check for overlaps with previous tier
		if i > 0 {
			prevTier := tiers[i-1]
			if tier.PageFrom <= prevTier.PageTo {
				return fmt.Errorf("tier '%s' page [%d-%d] overlaps with tier '%s' page [%d-%d]",
					tier.Name, tier.PageFrom, tier.PageTo,
					prevTier.Name, prevTier.PageFrom, prevTier.PageTo)
			}
		}
	}

	return nil
}

func (c *MarketsFetcherConfig) GetTTL() time.Duration {
	if c.TTL > 0 {
		return c.TTL
	}

	return 30 * time.Minute
}
