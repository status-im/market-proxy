package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCoingeckoPricesFetcher_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      PricesFetcherConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: PricesFetcherConfig{
				Tiers: []PriceTier{
					{
						Name:           "tier1",
						TokenFrom:      1,
						TokenTo:        1000,
						UpdateInterval: 30 * time.Second,
					},
					{
						Name:           "tier2",
						TokenFrom:      1001,
						TokenTo:        10000,
						UpdateInterval: 5 * time.Minute,
					},
				},
			},
			expectError: false,
		},
		{
			name: "no tiers",
			config: PricesFetcherConfig{
				Tiers: []PriceTier{},
			},
			expectError: true,
			errorMsg:    "at least one tier must be configured",
		},
		{
			name: "tier with empty name",
			config: PricesFetcherConfig{
				Tiers: []PriceTier{
					{
						Name:           "",
						TokenFrom:      1,
						TokenTo:        1000,
						UpdateInterval: 30 * time.Second,
					},
				},
			},
			expectError: true,
			errorMsg:    "name cannot be empty",
		},
		{
			name: "tier with invalid token_from",
			config: PricesFetcherConfig{
				Tiers: []PriceTier{
					{
						Name:           "tier1",
						TokenFrom:      0,
						TokenTo:        1000,
						UpdateInterval: 30 * time.Second,
					},
				},
			},
			expectError: true,
			errorMsg:    "token_from must be greater than 0",
		},
		{
			name: "tier with token_to < token_from",
			config: PricesFetcherConfig{
				Tiers: []PriceTier{
					{
						Name:           "tier1",
						TokenFrom:      1000,
						TokenTo:        100,
						UpdateInterval: 30 * time.Second,
					},
				},
			},
			expectError: true,
			errorMsg:    "token_to (100) must be >= token_from (1000)",
		},
		{
			name: "tier with zero update interval",
			config: PricesFetcherConfig{
				Tiers: []PriceTier{
					{
						Name:           "tier1",
						TokenFrom:      1,
						TokenTo:        1000,
						UpdateInterval: 0,
					},
				},
			},
			expectError: true,
			errorMsg:    "update_interval must be greater than 0",
		},
		{
			name: "overlapping tiers",
			config: PricesFetcherConfig{
				Tiers: []PriceTier{
					{
						Name:           "tier1",
						TokenFrom:      1,
						TokenTo:        1000,
						UpdateInterval: 30 * time.Second,
					},
					{
						Name:           "tier2",
						TokenFrom:      500,
						TokenTo:        1500,
						UpdateInterval: 5 * time.Minute,
					},
				},
			},
			expectError: true,
			errorMsg:    "overlaps with tier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCoingeckoPricesFetcher_GetTTL(t *testing.T) {
	tests := []struct {
		name     string
		config   PricesFetcherConfig
		expected time.Duration
	}{
		{
			name: "with custom TTL",
			config: PricesFetcherConfig{
				TTL: 1 * time.Minute,
			},
			expected: 1 * time.Minute,
		},
		{
			name: "with zero TTL (default)",
			config: PricesFetcherConfig{
				TTL: 0,
			},
			expected: 30 * time.Second,
		},
		{
			name: "with negative TTL (default)",
			config: PricesFetcherConfig{
				TTL: -1 * time.Second,
			},
			expected: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetTTL()
			assert.Equal(t, tt.expected, result)
		})
	}
}
