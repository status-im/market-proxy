package fetcher_by_id

import (
	"strings"
	"testing"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
)

func TestFetcherByIdConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  config.FetcherByIdConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid single mode config",
			config: config.FetcherByIdConfig{
				Name:           "coins",
				EndpointPath:   "/api/v3/coins/{{id}}",
				TTL:            24 * time.Hour,
				UpdateInterval: 30 * time.Minute,
				TopIdsLimit:    1000,
			},
			wantErr: false,
		},
		{
			name: "valid batch mode config",
			config: config.FetcherByIdConfig{
				Name:           "prices",
				EndpointPath:   "/api/v3/simple/price?ids={{ids_list}}",
				TTL:            10 * time.Minute,
				UpdateInterval: 30 * time.Second,
				TopIdsLimit:    500,
				ChunkSize:      100,
			},
			wantErr: false,
		},
		{
			name: "valid config with tiers",
			config: config.FetcherByIdConfig{
				Name:         "coins",
				EndpointPath: "/api/v3/coins/{{id}}",
				TTL:          24 * time.Hour,
				Tiers: []config.GenericTier{
					{
						Name:           "top-100",
						IdFrom:         1,
						IdTo:           100,
						UpdateInterval: 1 * time.Hour,
					},
					{
						Name:           "top-101-1000",
						IdFrom:         101,
						IdTo:           1000,
						UpdateInterval: 6 * time.Hour,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: config.FetcherByIdConfig{
				EndpointPath:   "/api/v3/coins/{{id}}",
				UpdateInterval: 30 * time.Minute,
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing endpoint_path",
			config: config.FetcherByIdConfig{
				Name:           "coins",
				UpdateInterval: 30 * time.Minute,
			},
			wantErr: true,
			errMsg:  "endpoint_path is required",
		},
		{
			name: "missing placeholder in endpoint",
			config: config.FetcherByIdConfig{
				Name:           "coins",
				EndpointPath:   "/api/v3/coins/bitcoin",
				UpdateInterval: 30 * time.Minute,
			},
			wantErr: true,
			errMsg:  "must contain either",
		},
		{
			name: "both placeholders in endpoint",
			config: config.FetcherByIdConfig{
				Name:           "coins",
				EndpointPath:   "/api/v3/coins/{{id}}?ids={{ids_list}}",
				UpdateInterval: 30 * time.Minute,
			},
			wantErr: true,
			errMsg:  "cannot contain both",
		},
		{
			name: "missing update_interval without tiers",
			config: config.FetcherByIdConfig{
				Name:         "coins",
				EndpointPath: "/api/v3/coins/{{id}}",
			},
			wantErr: true,
			errMsg:  "update_interval is required",
		},
		{
			name: "overlapping tiers",
			config: config.FetcherByIdConfig{
				Name:         "coins",
				EndpointPath: "/api/v3/coins/{{id}}",
				Tiers: []config.GenericTier{
					{
						Name:           "tier1",
						IdFrom:         1,
						IdTo:           100,
						UpdateInterval: 1 * time.Hour,
					},
					{
						Name:           "tier2",
						IdFrom:         50, // Overlaps with tier1
						IdTo:           200,
						UpdateInterval: 2 * time.Hour,
					},
				},
			},
			wantErr: true,
			errMsg:  "overlaps",
		},
		{
			name: "invalid tier range",
			config: config.FetcherByIdConfig{
				Name:         "coins",
				EndpointPath: "/api/v3/coins/{{id}}",
				Tiers: []config.GenericTier{
					{
						Name:           "tier1",
						IdFrom:         100,
						IdTo:           50, // Invalid: from > to
						UpdateInterval: 1 * time.Hour,
					},
				},
			},
			wantErr: true,
			errMsg:  "id_to",
		},
		{
			name: "tier with zero id_from",
			config: config.FetcherByIdConfig{
				Name:         "coins",
				EndpointPath: "/api/v3/coins/{{id}}",
				Tiers: []config.GenericTier{
					{
						Name:           "tier1",
						IdFrom:         0,
						IdTo:           100,
						UpdateInterval: 1 * time.Hour,
					},
				},
			},
			wantErr: true,
			errMsg:  "id_from must be greater than 0",
		},
		{
			name: "tier with empty name",
			config: config.FetcherByIdConfig{
				Name:         "coins",
				EndpointPath: "/api/v3/coins/{{id}}",
				Tiers: []config.GenericTier{
					{
						Name:           "",
						IdFrom:         1,
						IdTo:           100,
						UpdateInterval: 1 * time.Hour,
					},
				},
			},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name: "tier with zero update_interval",
			config: config.FetcherByIdConfig{
				Name:         "coins",
				EndpointPath: "/api/v3/coins/{{id}}",
				Tiers: []config.GenericTier{
					{
						Name:           "tier1",
						IdFrom:         1,
						IdTo:           100,
						UpdateInterval: 0,
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
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.True(t, strings.Contains(err.Error(), tt.errMsg),
						"expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFetcherByIdConfig_GetFetchMode(t *testing.T) {
	tests := []struct {
		name         string
		endpointPath string
		expectedMode config.FetchMode
	}{
		{
			name:         "single mode with {{id}}",
			endpointPath: "/api/v3/coins/{{id}}",
			expectedMode: config.FetchModeSingle,
		},
		{
			name:         "batch mode with {{ids_list}}",
			endpointPath: "/api/v3/simple/price?ids={{ids_list}}",
			expectedMode: config.FetchModeBatch,
		},
		{
			name:         "single mode by default",
			endpointPath: "/api/v3/coins/bitcoin",
			expectedMode: config.FetchModeSingle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.FetcherByIdConfig{
				EndpointPath: tt.endpointPath,
			}
			assert.Equal(t, tt.expectedMode, cfg.GetFetchMode())
		})
	}
}

func TestFetcherByIdConfig_IsBatchMode(t *testing.T) {
	singleCfg := &config.FetcherByIdConfig{
		EndpointPath: "/api/v3/coins/{{id}}",
	}
	assert.False(t, singleCfg.IsBatchMode())

	batchCfg := &config.FetcherByIdConfig{
		EndpointPath: "/api/v3/simple/price?ids={{ids_list}}",
	}
	assert.True(t, batchCfg.IsBatchMode())
}

func TestFetcherByIdConfig_HasTiers(t *testing.T) {
	noTiersCfg := &config.FetcherByIdConfig{
		Name: "test",
	}
	assert.False(t, noTiersCfg.HasTiers())

	withTiersCfg := &config.FetcherByIdConfig{
		Name: "test",
		Tiers: []config.GenericTier{
			{Name: "tier1", IdFrom: 1, IdTo: 100, UpdateInterval: time.Hour},
		},
	}
	assert.True(t, withTiersCfg.HasTiers())
}

func TestFetcherByIdConfig_GetChunkSize(t *testing.T) {
	// Default chunk size
	cfg1 := &config.FetcherByIdConfig{}
	assert.Equal(t, 100, cfg1.GetChunkSize())

	// Custom chunk size
	cfg2 := &config.FetcherByIdConfig{ChunkSize: 50}
	assert.Equal(t, 50, cfg2.GetChunkSize())

	// Negative chunk size returns default
	cfg3 := &config.FetcherByIdConfig{ChunkSize: -1}
	assert.Equal(t, 100, cfg3.GetChunkSize())
}

func TestFetcherByIdConfig_GetTTL(t *testing.T) {
	// Default TTL
	cfg1 := &config.FetcherByIdConfig{}
	assert.Equal(t, 5*time.Minute, cfg1.GetTTL())

	// Custom TTL
	cfg2 := &config.FetcherByIdConfig{TTL: 1 * time.Hour}
	assert.Equal(t, 1*time.Hour, cfg2.GetTTL())

	// Zero TTL returns default
	cfg3 := &config.FetcherByIdConfig{TTL: 0}
	assert.Equal(t, 5*time.Minute, cfg3.GetTTL())
}

func TestFetcherByIdConfig_GetCacheKeyPrefix(t *testing.T) {
	cfg := &config.FetcherByIdConfig{Name: "coins"}
	assert.Equal(t, "coins", cfg.GetCacheKeyPrefix())
}

func TestFetcherByIdConfig_BuildCacheKey(t *testing.T) {
	cfg := &config.FetcherByIdConfig{Name: "coins"}

	key1 := cfg.BuildCacheKey("bitcoin")
	assert.Equal(t, "coins:id:bitcoin", key1)

	key2 := cfg.BuildCacheKey("ethereum")
	assert.Equal(t, "coins:id:ethereum", key2)

	// Same ID should produce same key
	key3 := cfg.BuildCacheKey("bitcoin")
	assert.Equal(t, key1, key3)
}

func TestFetcherByIdConfig_GetMaxIdLimit(t *testing.T) {
	// With tiers
	cfgWithTiers := &config.FetcherByIdConfig{
		Tiers: []config.GenericTier{
			{Name: "tier1", IdFrom: 1, IdTo: 100, UpdateInterval: time.Hour},
			{Name: "tier2", IdFrom: 101, IdTo: 500, UpdateInterval: time.Hour},
		},
	}
	assert.Equal(t, 500, cfgWithTiers.GetMaxIdLimit())

	// Without tiers, with TopIdsLimit
	cfgWithLimit := &config.FetcherByIdConfig{
		TopIdsLimit: 1000,
	}
	assert.Equal(t, 1000, cfgWithLimit.GetMaxIdLimit())

	// Without tiers or limit
	cfgDefault := &config.FetcherByIdConfig{}
	assert.Equal(t, 1000, cfgDefault.GetMaxIdLimit())
}

func TestFetcherByIdConfig_BuildQueryParams(t *testing.T) {
	cfg := &config.FetcherByIdConfig{
		ParamsOverride: map[string]interface{}{
			"localization":  false,
			"tickers":       true,
			"market_data":   false,
			"vs_currencies": "usd,eur",
			"precision":     8,
			"float_value":   3.14,
			"int_as_float":  42.0,
		},
	}

	params := cfg.BuildQueryParams()

	assert.Equal(t, "false", params["localization"])
	assert.Equal(t, "true", params["tickers"])
	assert.Equal(t, "false", params["market_data"])
	assert.Equal(t, "usd,eur", params["vs_currencies"])
	assert.Equal(t, "8", params["precision"])
	assert.Equal(t, "3.14", params["float_value"])
	assert.Equal(t, "42", params["int_as_float"]) // Integer-like float should be formatted as int
}
