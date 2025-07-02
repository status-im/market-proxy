package coingecko_markets

import (
	"testing"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
)

// Helper functions for creating string and int pointers
func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }
func boolPtr(b bool) *bool       { return &b }

func TestService_getParamsOverride(t *testing.T) {
	t.Run("No normalization config - returns params as is", func(t *testing.T) {
		service := NewService(&MockCache{}, createTestConfig())

		originalParams := cg.MarketsParams{
			Currency:              "eur",
			Order:                 "volume_desc",
			PerPage:               100,
			SparklineEnabled:      true,
			PriceChangePercentage: []string{"7d", "30d"},
			Category:              "defi",
		}

		result := service.getParamsOverride(originalParams)

		assert.Equal(t, originalParams, result)
	})

	t.Run("With normalization config - overrides parameters", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.CoingeckoMarkets.MarketParamsNormalize = &config.MarketParamsNormalize{
			VsCurrency:            stringPtr("usd"),
			Order:                 stringPtr("market_cap_desc"),
			PerPage:               intPtr(250),
			Sparkline:             boolPtr(false),
			PriceChangePercentage: stringPtr("1h,24h"),
			Category:              stringPtr(""),
		}

		service := NewService(&MockCache{}, cfg)

		originalParams := cg.MarketsParams{
			Currency:              "eur",
			Order:                 "volume_desc",
			PerPage:               100,
			SparklineEnabled:      true,
			PriceChangePercentage: []string{"7d", "30d"},
			Category:              "defi",
		}

		result := service.getParamsOverride(originalParams)

		assert.Equal(t, "usd", result.Currency)
		assert.Equal(t, "market_cap_desc", result.Order)
		assert.Equal(t, 250, result.PerPage)
		assert.Equal(t, false, result.SparklineEnabled)
		assert.Equal(t, []string{"1h", "24h"}, result.PriceChangePercentage)
		assert.Equal(t, "", result.Category) // Category should be overridden to empty string
	})

	t.Run("Partial normalization config - overrides only configured parameters", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.CoingeckoMarkets.MarketParamsNormalize = &config.MarketParamsNormalize{
			VsCurrency: stringPtr("usd"),
			Order:      stringPtr("market_cap_desc"),
			// Other fields not configured
		}

		service := NewService(&MockCache{}, cfg)

		originalParams := cg.MarketsParams{
			Currency:              "eur",
			Order:                 "volume_desc",
			PerPage:               100,
			SparklineEnabled:      true,
			PriceChangePercentage: []string{"7d", "30d"},
			Category:              "defi",
		}

		result := service.getParamsOverride(originalParams)

		assert.Equal(t, "usd", result.Currency)                              // overridden
		assert.Equal(t, "market_cap_desc", result.Order)                     // overridden
		assert.Equal(t, 100, result.PerPage)                                 // not overridden
		assert.Equal(t, true, result.SparklineEnabled)                       // not overridden
		assert.Equal(t, []string{"7d", "30d"}, result.PriceChangePercentage) // not overridden
		assert.Equal(t, "defi", result.Category)                             // not overridden
	})
}
