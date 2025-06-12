package e2etest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/status-im/market-proxy/config"
)

// createTestConfig creates a test configuration and returns the path to the file
func createTestConfig(mockURL, mockWSURL string) (string, error) {
	// Create a temporary directory for configuration
	tempDir, err := os.MkdirTemp("", "market-proxy-test")
	if err != nil {
		return "", err
	}

	// Create configuration file content
	configContent := `
coingecko_leaderboard:
  update_interval: 1s       # shorter interval for tests
  limit: 20                 # fewer tokens for tests
  request_delay: 100ms      # short delay for tests

coingecko_prices:
  chunk_size: 100           # smaller chunks for tests
  request_delay: 100ms      # short delay for tests
  currencies:               # test currencies
    - usd
    - eur

coingecko_coinslist:
  update_interval: 1s       # shorter interval for tests
  supported_platforms: []   # array of supported blockchain platforms

tokens_file: "%s"           # path to tokens file will be inserted

# URLs for API (mock)
override_coingecko_public_url: "%s"  # URL for CoinGecko public API
override_coingecko_pro_url: "%s"     # URL for CoinGecko Pro API
override_binance_wsurl: "%s"         # URL for Binance WebSocket
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
	configContent = sprintf(configContent, tokensFilePath, mockURL, mockURL, mockWSURL)

	// Create configuration file
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	return configPath, nil
}

// loadTestConfig creates and loads test configuration
func loadTestConfig(mockURL, mockWSURL string) (*config.Config, string, error) {
	configPath, err := createTestConfig(mockURL, mockWSURL)
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
