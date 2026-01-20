package e2etest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/status-im/market-proxy/config"
)

// createTestConfig creates a test configuration and returns the path to the file
func createTestConfig(mockURL string) (string, error) {
	// Create a temporary directory for configuration
	tempDir, err := os.MkdirTemp("", "market-proxy-test")
	if err != nil {
		return "", err
	}

	// Create configuration file content
	configContent := `
coingecko_leaderboard:
  update_interval: 1s                 # shorter interval for tests
  top_markets_update_interval: 1s     # enable markets updates for tests
  top_markets_limit: 20               # fewer tokens for tests 
  request_delay: 100ms                # short delay for tests
  currency: "usd"                     # must match the markets service currency

coingecko_markets:
  request_delay: 100ms     # short delay for tests
  default_ttl: 5m          # default cache TTL for tests
  market_params_normalize: # normalize parameters for consistent caching
    vs_currency: "usd"     # always use USD for tests
    order: "market_cap_desc" # always order by market cap
    per_page: 50           # smaller page size for tests
    sparkline: false       # no sparkline for tests
    price_change_percentage: "1h,24h" # test price changes
    category: ""           # no category filtering
  tiers:                   # required tier configuration
    - name: "test"         # test tier
      page_from: 1        # tokens 1-2 for tests - much smaller range
      page_to: 2
      update_interval: 1s  # fast updates for tests
      ttl: 5s              # short TTL for tests

coingecko_prices:
  request_delay: 100ms      # short delay for tests
  currencies:               # test currencies
    - usd
    - eur
  tiers:                    # required tier configuration  
    - name: "top-1000"      # test tier for top tokens
      token_from: 1         # tokens 1-1000 for tests
      token_to: 1000
      update_interval: 1s   # fast update interval for tests
    - name: "top-1001-10000" # test tier for remaining tokens
      token_from: 1001      # tokens 1001-10000 for tests
      token_to: 10000
      update_interval: 2s   # fast update interval for tests

coingecko_coinslist:
  update_interval: 1s       # shorter interval for tests
  request_delay: 100ms      # short delay for tests
  supported_platforms: []   # array of supported blockchain platforms

coingecko_token_list:
  update_interval: 1s       # shorter interval for tests
  supported_platforms:
    - ethereum
    - base
    - arbitrum-one
    - optimistic-ethereum
    - linea
    - polygon-zkevm
    - unichain
    - katana
    - ink
    - abstract
    - zksync
    - soneium
    - scroll
    - blast
    - binance-smart-chain

coingecko_coins:
  name: "coins"
  endpoint_path: "/api/v3/coins/{{id}}"
  ttl: 1m
  params_override:
    localization: false
    tickers: false
    market_data: false
  tiers:
    - name: "test-tier"
      id_from: 1
      id_to: 10
      update_interval: 1s

tokens_file: "%s"           # path to tokens file will be inserted

# URLs for API (mock)
override_coingecko_public_url: "%s"  # URL for CoinGecko public API
override_coingecko_pro_url: "%s"     # URL for CoinGecko Pro API
`

	// Create tokens file
	tokensFilePath := filepath.Join(tempDir, "tokens.json")
	tokensContent := `
{
  "api_tokens": ["test-api-key"],
  "demo_api_tokens": ["test-demo-key"]
}
`

	if err := os.WriteFile(tokensFilePath, []byte(tokensContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Insert values into configuration
	configContent = sprintf(configContent, tokensFilePath, mockURL, mockURL)

	// Create configuration file
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	return configPath, nil
}

// loadTestConfig creates and loads test configuration
func loadTestConfig(mockURL string) (*config.Config, string, error) {
	configPath, err := createTestConfig(mockURL)
	if err != nil {
		return nil, "", err
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		os.RemoveAll(filepath.Dir(configPath))
		return nil, "", err
	}

	return cfg, configPath, nil
}

// cleanupTestConfig removes the temporary directory with configuration
func cleanupTestConfig(configPath string) {
	os.RemoveAll(filepath.Dir(configPath))
}

// sprintf - helper function for string formatting
func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
