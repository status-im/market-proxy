package config

import (
	"testing"
	"time"
)

func TestCoingeckoMarketsFetcher_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CoingeckoMarketsFetcher
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tier configuration",
			config: CoingeckoMarketsFetcher{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       1,
						PageTo:         500,
						UpdateInterval: 30 * time.Second,
					},
					{
						Name:           "tier2",
						PageFrom:       501,
						PageTo:         10000,
						UpdateInterval: 30 * time.Minute,
					},
				},
			},
			wantErr: false,
		},

		{
			name: "overlapping tiers",
			config: CoingeckoMarketsFetcher{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       1,
						PageTo:         500,
						UpdateInterval: 30 * time.Second,
					},
					{
						Name:           "tier2",
						PageFrom:       450, // Overlaps with tier1
						PageTo:         1000,
						UpdateInterval: 30 * time.Minute,
					},
				},
			},
			wantErr: true,
			errMsg:  "overlaps",
		},
		{
			name: "invalid range (from > to)",
			config: CoingeckoMarketsFetcher{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       500,
						PageTo:         100, // Invalid: from > to
						UpdateInterval: 30 * time.Second,
					},
				},
			},
			wantErr: true,
			errMsg:  "page_to",
		},
		{
			name: "zero page_from",
			config: CoingeckoMarketsFetcher{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       0, // Invalid: must be > 0
						PageTo:         500,
						UpdateInterval: 30 * time.Second,
					},
				},
			},
			wantErr: true,
			errMsg:  "page_from must be greater than 0",
		},
		{
			name: "empty tier name",
			config: CoingeckoMarketsFetcher{
				Tiers: []MarketTier{
					{
						Name:           "", // Invalid: empty name
						PageFrom:       1,
						PageTo:         500,
						UpdateInterval: 30 * time.Second,
					},
				},
			},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name: "zero update interval",
			config: CoingeckoMarketsFetcher{
				Tiers: []MarketTier{
					{
						Name:           "tier1",
						PageFrom:       1,
						PageTo:         500,
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
					t.Errorf("CoingeckoMarketsFetcher.Validate() expected error but got none")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("CoingeckoMarketsFetcher.Validate() error = %v, expected to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("CoingeckoMarketsFetcher.Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(substr) > 0 && len(s) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
