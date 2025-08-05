package config

import (
	"os"
	"testing"
	"time"
)

func createTestTokens(t *testing.T) string {
	tokens := `{
		"api_tokens": ["test-token-1", "test-token-2"]
	}`

	tmpfile, err := os.CreateTemp("", "tokens-*.json")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte(tokens)); err != nil {
		t.Fatal(err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}

// TestLoadConfig verifies the correct loading of updated configuration parameters
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		validateCfg func(*testing.T, *Config)
	}{
		{
			name: "valid config",
			configYAML: `
tokens_file: "test_tokens.json"
coingecko_leaderboard:
  top_markets_update_interval: 1m
  top_markets_limit: 100
  currency: usd
coingecko_markets:
  tiers:
    - name: "test"
      range_from: 1
      range_to: 100
      update_interval: 1m
      ttl: 5m
coingecko_coinslist:
  update_interval: 30m
  supported_platforms:
    - ethereum
    - polygon-pos
`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if cfg.TokensFile != "test_tokens.json" {
					t.Errorf("TokensFile = %v, want test_tokens.json", cfg.TokensFile)
				}
				if cfg.CoingeckoLeaderboard.TopMarketsUpdateInterval != time.Minute {
					t.Errorf("TopMarketsUpdateInterval = %v, want 1m", cfg.CoingeckoLeaderboard.TopMarketsUpdateInterval)
				}
				if cfg.CoingeckoLeaderboard.Currency != "usd" {
					t.Errorf("Currency = %v, want usd", cfg.CoingeckoLeaderboard.Currency)
				}
				if cfg.CoingeckoLeaderboard.TopMarketsLimit != 100 {
					t.Errorf("TopMarketsLimit = %v, want 100", cfg.CoingeckoLeaderboard.TopMarketsLimit)
				}
				if cfg.TokensFetcher.UpdateInterval != 30*time.Minute {
					t.Errorf("CoingeckoCoinslistFetcher.UpdateInterval = %v, want 30m", cfg.TokensFetcher.UpdateInterval)
				}
				if len(cfg.TokensFetcher.SupportedPlatforms) != 2 {
					t.Errorf("CoingeckoCoinslistFetcher.SupportedPlatforms length = %v, want 2", len(cfg.TokensFetcher.SupportedPlatforms))
				}
				if cfg.TokensFetcher.SupportedPlatforms[0] != "ethereum" || cfg.TokensFetcher.SupportedPlatforms[1] != "polygon-pos" {
					t.Errorf("CoingeckoCoinslistFetcher.SupportedPlatforms = %v, want [ethereum polygon-pos]", cfg.TokensFetcher.SupportedPlatforms)
				}
				if cfg.APITokens == nil {
					t.Error("APITokens should not be nil")
				} else {
					if len(cfg.APITokens.Tokens) != 0 {
						t.Errorf("Expected empty API tokens, got %d tokens", len(cfg.APITokens.Tokens))
					}
				}
			},
		},
		{
			name: "invalid yaml",
			configYAML: `
tokens_file: "test_tokens.json"
coingecko_leaderboard:
  top_markets_update_interval: invalid
  top_markets_limit: 100
coingecko_markets:
  tiers:
    - name: "test"
      range_from: 1
      range_to: 100
      update_interval: 1m
      ttl: 5m
`,
			wantErr: true,
		},
		{
			name: "missing required fields",
			configYAML: `
tokens_file: "test_tokens.json"
coingecko_leaderboard:
  top_markets_update_interval: 1m
coingecko_markets:
  tiers:
    - name: "test"
      range_from: 1
      range_to: 100
      update_interval: 1m
      ttl: 5m
`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if cfg.CoingeckoLeaderboard.TopMarketsLimit != 0 {
					t.Errorf("TopMarketsLimit should be empty, got %v", cfg.CoingeckoLeaderboard.TopMarketsLimit)
				}
			},
		},
		{
			name: "zero request delay",
			configYAML: `
tokens_file: "test_tokens.json"
coingecko_leaderboard:
  update_interval: 1m
  limit: 100
  request_delay: 0s
coingecko_markets:
  tiers:
    - name: "test"
      range_from: 1
      range_to: 100
      update_interval: 1m
      ttl: 5m
`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if cfg.CoingeckoLeaderboard.Currency != "" {
					t.Errorf("Currency = %v, want empty string", cfg.CoingeckoLeaderboard.Currency)
				}
			},
		},
		{
			name: "tokens fetcher config",
			configYAML: `
tokens_file: "test_tokens.json"
coingecko_leaderboard:
  top_markets_update_interval: 1m
  top_markets_limit: 100
coingecko_markets:
  tiers:
    - name: "test"
      range_from: 1
      range_to: 100
      update_interval: 1m
      ttl: 5m
coingecko_coinslist:
  update_interval: 30m
  supported_platforms:
    - ethereum
    - optimistic-ethereum
    - arbitrum-one
    - base
`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if cfg.TokensFetcher.UpdateInterval != 30*time.Minute {
					t.Errorf("CoingeckoCoinslistFetcher.UpdateInterval = %v, want 30m", cfg.TokensFetcher.UpdateInterval)
				}
				expectedPlatforms := []string{"ethereum", "optimistic-ethereum", "arbitrum-one", "base"}
				if len(cfg.TokensFetcher.SupportedPlatforms) != len(expectedPlatforms) {
					t.Errorf("CoingeckoCoinslistFetcher.SupportedPlatforms length = %v, want %v",
						len(cfg.TokensFetcher.SupportedPlatforms), len(expectedPlatforms))
				}
				for i, platform := range expectedPlatforms {
					if i < len(cfg.TokensFetcher.SupportedPlatforms) && cfg.TokensFetcher.SupportedPlatforms[i] != platform {
						t.Errorf("CoingeckoCoinslistFetcher.SupportedPlatforms[%d] = %v, want %v",
							i, cfg.TokensFetcher.SupportedPlatforms[i], platform)
					}
				}
			},
		},
		{
			name: "empty supported platforms",
			configYAML: `
tokens_file: "test_tokens.json"
coingecko_leaderboard:
  top_markets_update_interval: 1m
  top_markets_limit: 100
coingecko_markets:
  tiers:
    - name: "test"
      range_from: 1
      range_to: 100
      update_interval: 1m
      ttl: 5m
coingecko_coinslist:
  update_interval: 30m
  supported_platforms: []
`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if len(cfg.TokensFetcher.SupportedPlatforms) != 0 {
					t.Errorf("CoingeckoCoinslistFetcher.SupportedPlatforms should be empty, got %v", cfg.TokensFetcher.SupportedPlatforms)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.configYAML)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Load and validate config
			cfg, err := LoadConfig(tmpfile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validateCfg != nil {
				tt.validateCfg(t, cfg)
			}
		})
	}
}

func TestLoadAPITokens(t *testing.T) {
	tests := []struct {
		name       string
		tokensJSON string
		wantErr    bool
		validate   func(*testing.T, *APITokens)
	}{
		{
			name: "valid tokens",
			tokensJSON: `{
				"api_tokens": ["token1", "token2"]
			}`,
			wantErr: false,
			validate: func(t *testing.T, tokens *APITokens) {
				if len(tokens.Tokens) != 2 {
					t.Errorf("Tokens length = %v, want 2", len(tokens.Tokens))
				}
				if tokens.Tokens[0] != "token1" || tokens.Tokens[1] != "token2" {
					t.Errorf("Tokens = %v, want [token1 token2]", tokens.Tokens)
				}
			},
		},
		{
			name: "invalid json",
			tokensJSON: `{
				"api_tokens": ["token1", "token2"
			}`,
			wantErr: true,
		},
		{
			name: "empty tokens",
			tokensJSON: `{
				"api_tokens": []
			}`,
			wantErr: false,
			validate: func(t *testing.T, tokens *APITokens) {
				if len(tokens.Tokens) != 0 {
					t.Errorf("Tokens should be empty, got %v", tokens.Tokens)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary tokens file
			tmpfile, err := os.CreateTemp("", "tokens-*.json")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.tokensJSON)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Load and validate tokens
			tokens, err := LoadAPITokens(tmpfile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadAPITokens() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, tokens)
			}
		})
	}
}

func TestLoadConfigWithRealFiles(t *testing.T) {
	// Create both the config and tokens files first, so we can set the correct path
	// Create test tokens file
	tokensFile := createTestTokens(t)
	defer os.Remove(tokensFile)

	// Prepare content with correct tokens file path
	content := `
tokens_file: "` + tokensFile + `"
coingecko_leaderboard:
  top_markets_update_interval: 1m
  top_markets_limit: 100
  currency: usd
coingecko_markets:
  tiers:
    - name: "test"
      range_from: 1
      range_to: 100
      update_interval: 1m
      ttl: 5m
coingecko_coinslist:
  update_interval: 30m
  supported_platforms:
    - ethereum
    - optimistic-ethereum
    - arbitrum-one
`

	// Create test config file with correct tokens path
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Load config
	config, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate config
	if config.TokensFile != tokensFile {
		t.Errorf("TokensFile = %v, want %v", config.TokensFile, tokensFile)
	}

	// Validate APITokens were automatically loaded
	if config.APITokens == nil {
		t.Error("APITokens should not be nil")
	} else {
		if len(config.APITokens.Tokens) != 2 {
			t.Errorf("Expected 2 API tokens, got %d", len(config.APITokens.Tokens))
		}
		if config.APITokens.Tokens[0] != "test-token-1" || config.APITokens.Tokens[1] != "test-token-2" {
			t.Errorf("API tokens = %v, want [test-token-1 test-token-2]", config.APITokens.Tokens)
		}
	}
}
