package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfigFile writes a config YAML to a temp file and returns its path
func writeConfigFile(t *testing.T, configYAML string) string {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(tmpfile.Name()) })

	_, err = tmpfile.WriteString(configYAML)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	return tmpfile.Name()
}

// writeAndLoadConfig writes and loads a config YAML, failing the test on error
func writeAndLoadConfig(t *testing.T, configYAML string) *Config {
	t.Helper()

	cfg, err := LoadConfig(writeConfigFile(t, configYAML))
	require.NoError(t, err)

	return cfg
}

func TestCurrencyRatiosConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CurrencyRatiosConfig
		wantErr string
	}{
		{
			name:   "valid",
			config: GetDefaultCurrencyRatiosConfig(),
		},
		{
			name: "missing update interval",
			config: CurrencyRatiosConfig{
				ReferenceCoins: []string{"bitcoin"},
				Currencies:     []string{"usd"},
			},
			wantErr: "update_interval",
		},
		{
			name: "no reference coins",
			config: CurrencyRatiosConfig{
				UpdateInterval: time.Minute,
				Currencies:     []string{"usd"},
			},
			wantErr: "reference coin",
		},
		{
			name: "no currencies",
			config: CurrencyRatiosConfig{
				UpdateInterval: time.Minute,
				ReferenceCoins: []string{"bitcoin"},
			},
			wantErr: "currency must be configured",
		},
		{
			name: "base currency missing from currencies",
			config: CurrencyRatiosConfig{
				UpdateInterval: time.Minute,
				ReferenceCoins: []string{"bitcoin"},
				Currencies:     []string{"eur", "btc"},
			},
			wantErr: "base currency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestCurrencyRatiosConfig_IsEmpty(t *testing.T) {
	assert.True(t, (&CurrencyRatiosConfig{}).IsEmpty())

	defaults := GetDefaultCurrencyRatiosConfig()
	assert.False(t, defaults.IsEmpty())
}

func TestLoadConfig_CurrencyRatiosAndExchangeRates(t *testing.T) {
	tokensFile := createTestTokens(t)

	configYAML := `
tokens_file: "` + tokensFile + `"
coingecko_markets:
  tiers:
    - name: "test"
      page_from: 1
      page_to: 2
      update_interval: 1m
      ttl: 5m
currency_ratios:
  update_interval: 30s
  reference_coins:
    - bitcoin
    - ethereum
  currencies:
    - usd
    - eur
    - btc
coingecko_exchange_rates:
  update_interval: 10m
`

	cfg := writeAndLoadConfig(t, configYAML)

	assert.Equal(t, 30*time.Second, cfg.CurrencyRatios.UpdateInterval)
	assert.Equal(t, []string{"bitcoin", "ethereum"}, cfg.CurrencyRatios.ReferenceCoins)
	assert.Equal(t, []string{"usd", "eur", "btc"}, cfg.CurrencyRatios.Currencies)
	assert.Equal(t, 10*time.Minute, cfg.CoingeckoExchangeRates.UpdateInterval)
}

func TestLoadConfig_CurrencyRatiosDefaultsWhenAbsent(t *testing.T) {
	tokensFile := createTestTokens(t)

	configYAML := `
tokens_file: "` + tokensFile + `"
coingecko_markets:
  tiers:
    - name: "test"
      page_from: 1
      page_to: 2
      update_interval: 1m
      ttl: 5m
`

	cfg := writeAndLoadConfig(t, configYAML)

	assert.Equal(t, GetDefaultCurrencyRatiosConfig(), cfg.CurrencyRatios)
	assert.Equal(t, GetDefaultExchangeRatesConfig(), cfg.CoingeckoExchangeRates)
}

func TestLoadConfig_InvalidCurrencyRatiosIsRejected(t *testing.T) {
	tokensFile := createTestTokens(t)

	configYAML := `
tokens_file: "` + tokensFile + `"
coingecko_markets:
  tiers:
    - name: "test"
      page_from: 1
      page_to: 2
      update_interval: 1m
      ttl: 5m
currency_ratios:
  update_interval: 1m
  reference_coins:
    - bitcoin
  currencies:
    - eur
`

	path := writeConfigFile(t, configYAML)
	_, err := LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid currency_ratios configuration")
}
