package config

import (
	"strings"
	"testing"
	"time"
)

func TestCoingeckoMarketsFetcher_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  MarketsFetcherConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tier configuration",
			config: MarketsFetcherConfig{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       1,
						PageTo:         2, // optimized from 500
						UpdateInterval: 30 * time.Second,
					},
					{
						Name:           "tier2",
						PageFrom:       3, // adjusted to not overlap
						PageTo:         5, // optimized from 10000
						UpdateInterval: 30 * time.Minute,
					},
				},
			},
			wantErr: false,
		},

		{
			name: "overlapping tiers",
			config: MarketsFetcherConfig{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       1,
						PageTo:         3, // optimized from 500
						UpdateInterval: 30 * time.Second,
					},
					{
						Name:           "tier2",
						PageFrom:       2, // Overlaps with tier1 (still tests overlap logic)
						PageTo:         5, // optimized from 1000
						UpdateInterval: 30 * time.Minute,
					},
				},
			},
			wantErr: true,
			errMsg:  "overlaps",
		},
		{
			name: "invalid range (from > to)",
			config: MarketsFetcherConfig{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       3,
						PageTo:         1, // Invalid: from > to (still tests validation logic)
						UpdateInterval: 30 * time.Second,
					},
				},
			},
			wantErr: true,
			errMsg:  "page_to",
		},
		{
			name: "zero page_from",
			config: MarketsFetcherConfig{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       0, // Invalid: must be > 0
						PageTo:         2, // optimized from 500
						UpdateInterval: 30 * time.Second,
					},
				},
			},
			wantErr: true,
			errMsg:  "page_from must be greater than 0",
		},
		{
			name: "empty tier name",
			config: MarketsFetcherConfig{
				Tiers: []MarketTier{
					{
						Name:           "", // Invalid: empty name
						PageFrom:       1,
						PageTo:         2, // optimized from 500
						UpdateInterval: 30 * time.Second,
					},
				},
			},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name: "zero update interval",
			config: MarketsFetcherConfig{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       1,
						PageTo:         2, // optimized from 500
						UpdateInterval: 0, // Invalid: must be > 0

					},
				},
			},
			wantErr: true,
			errMsg:  "update_interval must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("MarketsFetcherConfig.Validate() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MarketsFetcherConfig.Validate() error = %v, expected to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("MarketsFetcherConfig.Validate() unexpected error = %v", err)
				}
			}
		})
	}
}
