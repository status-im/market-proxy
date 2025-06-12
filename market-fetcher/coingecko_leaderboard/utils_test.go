package coingecko_leaderboard

import (
	"testing"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
)

func TestConvertPriceResponseToPriceQuotes(t *testing.T) {
	t.Run("Valid conversion with all fields", func(t *testing.T) {
		priceResponse := cg.SimplePriceResponse{
			"bitcoin": map[string]interface{}{
				"usd":            50000.0,
				"usd_market_cap": 950000000000.0,
				"usd_24h_vol":    25000000000.0,
				"usd_24h_change": 2.5,
			},
			"ethereum": map[string]interface{}{
				"usd":            3000.0,
				"usd_market_cap": 360000000000.0,
				"usd_24h_vol":    15000000000.0,
				"usd_24h_change": -1.2,
			},
		}

		result := ConvertPriceResponseToPriceQuotes(priceResponse, "usd")

		assert.Len(t, result, 2)

		// Check bitcoin
		bitcoinQuote := result["bitcoin"]
		assert.Equal(t, 50000.0, bitcoinQuote.Price)
		assert.Equal(t, 950000000000.0, bitcoinQuote.MarketCap)
		assert.Equal(t, 25000000000.0, bitcoinQuote.Volume24h)
		assert.Equal(t, 2.5, bitcoinQuote.PercentChange24h)

		// Check ethereum
		ethQuote := result["ethereum"]
		assert.Equal(t, 3000.0, ethQuote.Price)
		assert.Equal(t, 360000000000.0, ethQuote.MarketCap)
		assert.Equal(t, 15000000000.0, ethQuote.Volume24h)
		assert.Equal(t, -1.2, ethQuote.PercentChange24h)
	})

	t.Run("Conversion with missing optional fields", func(t *testing.T) {
		priceResponse := cg.SimplePriceResponse{
			"bitcoin": map[string]interface{}{
				"usd": 50000.0,
				// Missing market cap, volume, and change
			},
			"ethereum": map[string]interface{}{
				"usd":            3000.0,
				"usd_market_cap": 360000000000.0,
				// Missing volume and change
			},
		}

		result := ConvertPriceResponseToPriceQuotes(priceResponse, "usd")

		assert.Len(t, result, 2)

		// Check bitcoin - only price should be set
		bitcoinQuote := result["bitcoin"]
		assert.Equal(t, 50000.0, bitcoinQuote.Price)
		assert.Equal(t, 0.0, bitcoinQuote.MarketCap)
		assert.Equal(t, 0.0, bitcoinQuote.Volume24h)
		assert.Equal(t, 0.0, bitcoinQuote.PercentChange24h)

		// Check ethereum - price and market cap should be set
		ethQuote := result["ethereum"]
		assert.Equal(t, 3000.0, ethQuote.Price)
		assert.Equal(t, 360000000000.0, ethQuote.MarketCap)
		assert.Equal(t, 0.0, ethQuote.Volume24h)
		assert.Equal(t, 0.0, ethQuote.PercentChange24h)
	})

	t.Run("Skip tokens without valid price", func(t *testing.T) {
		priceResponse := cg.SimplePriceResponse{
			"bitcoin": map[string]interface{}{
				"usd": 50000.0, // Valid price
			},
			"token_without_price": map[string]interface{}{
				"usd_market_cap": 1000000.0,
				// Missing price field
			},
			"token_with_zero_price": map[string]interface{}{
				"usd": 0.0, // Zero price - should be skipped
			},
			"token_with_negative_price": map[string]interface{}{
				"usd": -100.0, // Negative price - should be skipped
			},
			"token_with_invalid_price_type": map[string]interface{}{
				"usd": "invalid", // Wrong type - should be skipped
			},
		}

		result := ConvertPriceResponseToPriceQuotes(priceResponse, "usd")

		// Only bitcoin should be included
		assert.Len(t, result, 1)
		assert.Contains(t, result, "bitcoin")
		assert.Equal(t, 50000.0, result["bitcoin"].Price)
	})

	t.Run("Empty response", func(t *testing.T) {
		priceResponse := cg.SimplePriceResponse{}

		result := ConvertPriceResponseToPriceQuotes(priceResponse, "usd")

		assert.Len(t, result, 0)
	})

	t.Run("Different currency", func(t *testing.T) {
		priceResponse := cg.SimplePriceResponse{
			"bitcoin": map[string]interface{}{
				"eur":            42000.0,
				"eur_market_cap": 798000000000.0,
				"eur_24h_vol":    21000000000.0,
				"eur_24h_change": 1.8,
				"usd":            50000.0, // This should be ignored
			},
		}

		result := ConvertPriceResponseToPriceQuotes(priceResponse, "eur")

		assert.Len(t, result, 1)
		bitcoinQuote := result["bitcoin"]
		assert.Equal(t, 42000.0, bitcoinQuote.Price)
		assert.Equal(t, 798000000000.0, bitcoinQuote.MarketCap)
		assert.Equal(t, 21000000000.0, bitcoinQuote.Volume24h)
		assert.Equal(t, 1.8, bitcoinQuote.PercentChange24h)
	})

	t.Run("Invalid token data structure", func(t *testing.T) {
		priceResponse := cg.SimplePriceResponse{
			"bitcoin": map[string]interface{}{
				"usd": 50000.0,
			},
			"invalid_token":   "not_a_map",       // Invalid structure - should be skipped
			"another_invalid": []string{"array"}, // Invalid structure - should be skipped
		}

		result := ConvertPriceResponseToPriceQuotes(priceResponse, "usd")

		// Only bitcoin should be included
		assert.Len(t, result, 1)
		assert.Contains(t, result, "bitcoin")
		assert.Equal(t, 50000.0, result["bitcoin"].Price)
	})

	t.Run("Non-numeric field values", func(t *testing.T) {
		priceResponse := cg.SimplePriceResponse{
			"bitcoin": map[string]interface{}{
				"usd":            50000.0,
				"usd_market_cap": "invalid_market_cap", // Wrong type - should be ignored
				"usd_24h_vol":    true,                 // Wrong type - should be ignored
				"usd_24h_change": []int{1, 2, 3},       // Wrong type - should be ignored
			},
		}

		result := ConvertPriceResponseToPriceQuotes(priceResponse, "usd")

		assert.Len(t, result, 1)
		bitcoinQuote := result["bitcoin"]
		assert.Equal(t, 50000.0, bitcoinQuote.Price)
		assert.Equal(t, 0.0, bitcoinQuote.MarketCap)        // Should be 0 due to invalid type
		assert.Equal(t, 0.0, bitcoinQuote.Volume24h)        // Should be 0 due to invalid type
		assert.Equal(t, 0.0, bitcoinQuote.PercentChange24h) // Should be 0 due to invalid type
	})
}

func TestTopPricesUpdater_SetGetTopTokenIDs(t *testing.T) {
	cfg := &config.Config{}
	updater := NewTopPricesUpdater(cfg, nil)

	t.Run("Set and get token IDs", func(t *testing.T) {
		tokenIDs := []string{"bitcoin", "ethereum", "cardano"}
		updater.SetTopTokenIDs(tokenIDs)

		result := updater.getTopTokenIDs()
		assert.Equal(t, tokenIDs, result)
	})

	t.Run("Empty token IDs", func(t *testing.T) {
		updater.SetTopTokenIDs([]string{})

		result := updater.getTopTokenIDs()
		assert.Nil(t, result)
	})

	t.Run("Nil token IDs", func(t *testing.T) {
		updater.SetTopTokenIDs(nil)

		result := updater.getTopTokenIDs()
		assert.Nil(t, result)
	})

	t.Run("Overwrite existing token IDs", func(t *testing.T) {
		// Set initial token IDs
		initialIDs := []string{"bitcoin", "ethereum"}
		updater.SetTopTokenIDs(initialIDs)

		// Verify they're set
		result := updater.getTopTokenIDs()
		assert.Equal(t, initialIDs, result)

		// Overwrite with new IDs
		newIDs := []string{"cardano", "solana", "polkadot"}
		updater.SetTopTokenIDs(newIDs)

		// Verify they're updated
		result = updater.getTopTokenIDs()
		assert.Equal(t, newIDs, result)
	})

	t.Run("Returned slice is independent copy", func(t *testing.T) {
		originalIDs := []string{"bitcoin", "ethereum"}
		updater.SetTopTokenIDs(originalIDs)

		// Get the IDs and modify the returned slice
		result := updater.getTopTokenIDs()
		result[0] = "modified"

		// Original should be unchanged
		originalResult := updater.getTopTokenIDs()
		assert.Equal(t, "bitcoin", originalResult[0])
		assert.NotEqual(t, "modified", originalResult[0])
	})
}
