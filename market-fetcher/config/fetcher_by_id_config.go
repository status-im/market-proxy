package config

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// FetchMode represents the mode of fetching data
type FetchMode string

const (
	// FetchModeSingle fetches one item per request using {{id}} template
	FetchModeSingle FetchMode = "single"
	// FetchModeBatch fetches multiple items per request using {{ids_list}} template
	FetchModeBatch FetchMode = "batch"
)

// Template placeholders
const (
	TemplatePlaceholderID      = "{{id}}"
	TemplatePlaceholderIDsList = "{{ids_list}}"
)

// GenericTier defines a tier configuration for token ranges
type GenericTier struct {
	// Name of the tier (e.g., "top-500", "top-501-5000")
	Name string `yaml:"name"`

	// IdFrom is the start of token range (1-based, inclusive)
	IdFrom int `yaml:"id_from"`

	// IdTo is the end of token range (inclusive)
	IdTo int `yaml:"id_to"`

	// UpdateInterval is how often to refresh data for this tier
	UpdateInterval time.Duration `yaml:"update_interval"`

	// FetchCoinslistIds enables fetching missing tokens from coinslist
	FetchCoinslistIds bool `yaml:"fetch_coinslist_ids"`
}

// FetcherByIdConfig represents configuration for a generic CoinGecko fetcher
type FetcherByIdConfig struct {
	// Name is the identifier for this fetcher (e.g., "coins", "token_info")
	Name string `yaml:"name"`

	// EndpointPath is the URL path template with placeholders
	// Use {{id}} for single-ID mode: /api/v3/coins/{{id}}
	// Use {{ids_list}} for batch mode: /api/v3/simple/price?ids={{ids_list}}
	EndpointPath string `yaml:"endpoint_path"`

	// TTL is the time-to-live for cached data
	TTL time.Duration `yaml:"ttl"`

	// UpdateInterval is how often to refresh the data (used when no tiers configured)
	UpdateInterval time.Duration `yaml:"update_interval"`

	// TopIdsLimit is the maximum number of IDs to fetch (from top markets)
	// Used when no tiers configured
	TopIdsLimit int `yaml:"top_ids_limit"`

	// ChunkSize is the number of IDs per request in batch mode (default: 100)
	ChunkSize int `yaml:"chunk_size"`

	// ParamsOverride are query parameters for API requests
	ParamsOverride map[string]interface{} `yaml:"params_override"`

	// Tiers defines tier-based configuration for different token ranges
	// When configured, UpdateInterval and TopIdsLimit are ignored
	Tiers []GenericTier `yaml:"tiers"`
}

// GetFetchMode determines the fetch mode based on the endpoint path template
func (c *FetcherByIdConfig) GetFetchMode() FetchMode {
	if strings.Contains(c.EndpointPath, TemplatePlaceholderIDsList) {
		return FetchModeBatch
	}
	return FetchModeSingle
}

// IsBatchMode returns true if the fetcher operates in batch mode
func (c *FetcherByIdConfig) IsBatchMode() bool {
	return c.GetFetchMode() == FetchModeBatch
}

// HasTiers returns true if tier-based configuration is enabled
func (c *FetcherByIdConfig) HasTiers() bool {
	return len(c.Tiers) > 0
}

// GetChunkSize returns the chunk size with a default value
func (c *FetcherByIdConfig) GetChunkSize() int {
	if c.ChunkSize <= 0 {
		return 100 // default chunk size
	}
	return c.ChunkSize
}

// GetTTL returns the TTL with a default value
func (c *FetcherByIdConfig) GetTTL() time.Duration {
	if c.TTL <= 0 {
		return 5 * time.Minute // default TTL
	}
	return c.TTL
}

// GetCacheKeyPrefix returns the cache key prefix (always uses Name)
func (c *FetcherByIdConfig) GetCacheKeyPrefix() string {
	return c.Name
}

// BuildCacheKey creates a cache key for the given ID
func (c *FetcherByIdConfig) BuildCacheKey(id string) string {
	return fmt.Sprintf("%s:id:%s", c.GetCacheKeyPrefix(), id)
}

// GetMaxIdLimit returns the maximum ID limit from tiers or TopIdsLimit
func (c *FetcherByIdConfig) GetMaxIdLimit() int {
	if c.HasTiers() {
		maxTo := 0
		for _, tier := range c.Tiers {
			if tier.IdTo > maxTo {
				maxTo = tier.IdTo
			}
		}
		return maxTo
	}
	if c.TopIdsLimit > 0 {
		return c.TopIdsLimit
	}
	return 1000 // default
}

// Validate checks if the configuration is valid
func (c *FetcherByIdConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if c.EndpointPath == "" {
		return fmt.Errorf("endpoint_path is required")
	}

	// Check that endpoint has at least one placeholder
	hasSinglePlaceholder := strings.Contains(c.EndpointPath, TemplatePlaceholderID)
	hasBatchPlaceholder := strings.Contains(c.EndpointPath, TemplatePlaceholderIDsList)

	if !hasSinglePlaceholder && !hasBatchPlaceholder {
		return fmt.Errorf("endpoint_path must contain either %s or %s placeholder",
			TemplatePlaceholderID, TemplatePlaceholderIDsList)
	}

	// Cannot have both placeholders
	if hasSinglePlaceholder && hasBatchPlaceholder {
		return fmt.Errorf("endpoint_path cannot contain both %s and %s placeholders",
			TemplatePlaceholderID, TemplatePlaceholderIDsList)
	}

	// Validate tiers if configured
	if c.HasTiers() {
		if err := c.validateTiers(); err != nil {
			return fmt.Errorf("tier configuration validation failed: %w", err)
		}
	} else {
		// When no tiers, UpdateInterval is required
		if c.UpdateInterval <= 0 {
			return fmt.Errorf("update_interval is required when no tiers configured")
		}
	}

	return nil
}

// validateTiers validates that tier ranges don't overlap and are valid
func (c *FetcherByIdConfig) validateTiers() error {
	// Create a copy of tiers and sort by IdFrom for easier validation
	tiers := make([]GenericTier, len(c.Tiers))
	copy(tiers, c.Tiers)
	sort.Slice(tiers, func(i, j int) bool {
		return tiers[i].IdFrom < tiers[j].IdFrom
	})

	for i, tier := range tiers {
		// Validate individual tier
		if tier.Name == "" {
			return fmt.Errorf("tier at index %d: name cannot be empty", i)
		}
		if tier.IdFrom <= 0 {
			return fmt.Errorf("tier '%s': id_from must be greater than 0, got %d", tier.Name, tier.IdFrom)
		}
		if tier.IdTo < tier.IdFrom {
			return fmt.Errorf("tier '%s': id_to (%d) must be >= id_from (%d)", tier.Name, tier.IdTo, tier.IdFrom)
		}
		if tier.UpdateInterval <= 0 {
			return fmt.Errorf("tier '%s': update_interval must be greater than 0", tier.Name)
		}

		// Check for overlaps with previous tier
		if i > 0 {
			prevTier := tiers[i-1]
			if tier.IdFrom <= prevTier.IdTo {
				return fmt.Errorf("tier '%s' range [%d-%d] overlaps with tier '%s' range [%d-%d]",
					tier.Name, tier.IdFrom, tier.IdTo,
					prevTier.Name, prevTier.IdFrom, prevTier.IdTo)
			}
		}
	}

	return nil
}

// BuildQueryParams returns query parameters as string map
func (c *FetcherByIdConfig) BuildQueryParams() map[string]string {
	result := make(map[string]string)

	for key, value := range c.ParamsOverride {
		result[key] = formatParamValue(value)
	}

	return result
}

// formatParamValue converts a value to string for URL query
func formatParamValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		// Check if it's actually an integer value
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case []string:
		return strings.Join(v, ",")
	case []interface{}:
		// Handle YAML arrays which come as []interface{}
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprintf("%v", v)
	}
}
