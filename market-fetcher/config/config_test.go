package config

import (
	"os"
	"testing"
)

func createTestConfig(t *testing.T) string {
	content := `
coingecko_fetcher:
  update_interval_ms: 60000
  tokens_file: "test_tokens.json"
  limit: 100
  request_delay_ms: 1000
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}

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
coingecko_fetcher:
  update_interval_ms: 60000
  tokens_file: "test_tokens.json"
  limit: 100
  request_delay_ms: 1000
`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if cfg.CoinGeckoFetcher.UpdateIntervalMs != 60000 {
					t.Errorf("UpdateIntervalMs = %v, want 60000", cfg.CoinGeckoFetcher.UpdateIntervalMs)
				}
				if cfg.CoinGeckoFetcher.RequestDelayMs != 1000 {
					t.Errorf("RequestDelayMs = %v, want 1000", cfg.CoinGeckoFetcher.RequestDelayMs)
				}
				if cfg.CoinGeckoFetcher.Limit != 100 {
					t.Errorf("Limit = %v, want 100", cfg.CoinGeckoFetcher.Limit)
				}
			},
		},
		{
			name: "invalid yaml",
			configYAML: `
coingecko_fetcher:
  update_interval_ms: invalid
  tokens_file: "test_tokens.json"
  limit: 100
`,
			wantErr: true,
		},
		{
			name: "missing required fields",
			configYAML: `
coingecko_fetcher:
  update_interval_ms: 60000
  tokens_file: "test_tokens.json"
`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if cfg.CoinGeckoFetcher.Limit != 0 {
					t.Errorf("Limit should be empty, got %v", cfg.CoinGeckoFetcher.Limit)
				}
			},
		},
		{
			name: "zero request delay",
			configYAML: `
coingecko_fetcher:
  update_interval_ms: 60000
  tokens_file: "test_tokens.json"
  limit: 100
  request_delay_ms: 0
`,
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if cfg.CoinGeckoFetcher.RequestDelayMs != 0 {
					t.Errorf("RequestDelayMs = %v, want 0", cfg.CoinGeckoFetcher.RequestDelayMs)
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
	// Create test config file
	configFile := createTestConfig(t)
	defer os.Remove(configFile)

	// Create test tokens file
	tokensFile := createTestTokens(t)
	defer os.Remove(tokensFile)

	// Update config to point to the test tokens file
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate config
	if config.CoinGeckoFetcher.UpdateIntervalMs != 60000 {
		t.Errorf("UpdateIntervalMs = %v, want 60000", config.CoinGeckoFetcher.UpdateIntervalMs)
	}
	if config.CoinGeckoFetcher.RequestDelayMs != 1000 {
		t.Errorf("RequestDelayMs = %v, want 1000", config.CoinGeckoFetcher.RequestDelayMs)
	}
	if config.CoinGeckoFetcher.Limit != 100 {
		t.Errorf("Limit = %v, want 100", config.CoinGeckoFetcher.Limit)
	}

	// Load and validate tokens
	tokens, err := LoadAPITokens(tokensFile)
	if err != nil {
		t.Fatalf("Failed to load tokens: %v", err)
	}

	if len(tokens.Tokens) != 2 {
		t.Errorf("Tokens length = %v, want 2", len(tokens.Tokens))
	}
	if tokens.Tokens[0] != "test-token-1" || tokens.Tokens[1] != "test-token-2" {
		t.Errorf("Tokens = %v, want [test-token-1 test-token-2]", tokens.Tokens)
	}
}
