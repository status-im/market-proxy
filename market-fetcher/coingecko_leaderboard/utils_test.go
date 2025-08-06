package coingecko_leaderboard

import (
	"testing"

	cg "github.com/status-im/market-proxy/interfaces"

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

func TestConvertMarketsResponseToCoinData(t *testing.T) {
	t.Run("Valid conversion with all fields", func(t *testing.T) {
		marketsData := []interface{}{
			map[string]interface{}{
				"id":                          "bitcoin",
				"symbol":                      "btc",
				"name":                        "Bitcoin",
				"image":                       "https://coin-images.coingecko.com/coins/images/1/large/bitcoin.png",
				"current_price":               50000.0,
				"market_cap":                  950000000000.0,
				"total_volume":                25000000000.0,
				"price_change_percentage_24h": 2.5,
			},
			map[string]interface{}{
				"id":                          "ethereum",
				"symbol":                      "eth",
				"name":                        "Ethereum",
				"image":                       "https://coin-images.coingecko.com/coins/images/279/large/ethereum.png",
				"current_price":               3000.0,
				"market_cap":                  360000000000.0,
				"total_volume":                15000000000.0,
				"price_change_percentage_24h": -1.2,
			},
		}

		result := ConvertMarketsResponseToCoinData(marketsData)

		assert.Len(t, result, 2)

		// Check bitcoin
		bitcoin := result[0]
		assert.Equal(t, "bitcoin", bitcoin.ID)
		assert.Equal(t, "btc", bitcoin.Symbol)
		assert.Equal(t, "Bitcoin", bitcoin.Name)
		assert.Equal(t, "https://coin-images.coingecko.com/coins/images/1/large/bitcoin.png", bitcoin.Image)
		assert.Equal(t, 50000.0, bitcoin.CurrentPrice)
		assert.Equal(t, 950000000000.0, bitcoin.MarketCap)
		assert.Equal(t, 25000000000.0, bitcoin.TotalVolume)
		assert.Equal(t, 2.5, bitcoin.PriceChangePercentage24h)

		// Check ethereum
		ethereum := result[1]
		assert.Equal(t, "ethereum", ethereum.ID)
		assert.Equal(t, "eth", ethereum.Symbol)
		assert.Equal(t, "Ethereum", ethereum.Name)
		assert.Equal(t, "https://coin-images.coingecko.com/coins/images/279/large/ethereum.png", ethereum.Image)
		assert.Equal(t, 3000.0, ethereum.CurrentPrice)
		assert.Equal(t, 360000000000.0, ethereum.MarketCap)
		assert.Equal(t, 15000000000.0, ethereum.TotalVolume)
		assert.Equal(t, -1.2, ethereum.PriceChangePercentage24h)
	})

	t.Run("Conversion with missing fields", func(t *testing.T) {
		marketsData := []interface{}{
			map[string]interface{}{
				"id":     "bitcoin",
				"symbol": "btc",
				"name":   "Bitcoin",
				// Missing image, current_price, market_cap, total_volume, price_change_percentage_24h
			},
			map[string]interface{}{
				"id":            "ethereum",
				"symbol":        "eth",
				"name":          "Ethereum",
				"current_price": 3000.0,
				"market_cap":    360000000000.0,
				// Missing image, total_volume, price_change_percentage_24h
			},
		}

		result := ConvertMarketsResponseToCoinData(marketsData)

		assert.Len(t, result, 2)

		// Check bitcoin - only basic fields should be set
		bitcoin := result[0]
		assert.Equal(t, "bitcoin", bitcoin.ID)
		assert.Equal(t, "btc", bitcoin.Symbol)
		assert.Equal(t, "Bitcoin", bitcoin.Name)
		assert.Equal(t, "", bitcoin.Image)
		assert.Equal(t, 0.0, bitcoin.CurrentPrice)
		assert.Equal(t, 0.0, bitcoin.MarketCap)
		assert.Equal(t, 0.0, bitcoin.TotalVolume)
		assert.Equal(t, 0.0, bitcoin.PriceChangePercentage24h)

		// Check ethereum - partial fields should be set
		ethereum := result[1]
		assert.Equal(t, "ethereum", ethereum.ID)
		assert.Equal(t, "eth", ethereum.Symbol)
		assert.Equal(t, "Ethereum", ethereum.Name)
		assert.Equal(t, "", ethereum.Image)
		assert.Equal(t, 3000.0, ethereum.CurrentPrice)
		assert.Equal(t, 360000000000.0, ethereum.MarketCap)
		assert.Equal(t, 0.0, ethereum.TotalVolume)
		assert.Equal(t, 0.0, ethereum.PriceChangePercentage24h)
	})

	t.Run("Skip invalid items", func(t *testing.T) {
		marketsData := []interface{}{
			map[string]interface{}{
				"id":     "bitcoin",
				"symbol": "btc",
				"name":   "Bitcoin",
			},
			"invalid_string",             // Should be skipped
			[]string{"invalid", "array"}, // Should be skipped
			42,                           // Should be skipped
			map[string]interface{}{
				"id":     "ethereum",
				"symbol": "eth",
				"name":   "Ethereum",
			},
			nil, // Should be skipped
		}

		result := ConvertMarketsResponseToCoinData(marketsData)

		// Only bitcoin and ethereum should be included
		assert.Len(t, result, 2)
		assert.Equal(t, "bitcoin", result[0].ID)
		assert.Equal(t, "ethereum", result[1].ID)
	})

	t.Run("Empty input", func(t *testing.T) {
		marketsData := []interface{}{}

		result := ConvertMarketsResponseToCoinData(marketsData)

		assert.Len(t, result, 0)
		assert.NotNil(t, result) // Should return empty slice, not nil
	})

	t.Run("Nil input", func(t *testing.T) {
		result := ConvertMarketsResponseToCoinData(nil)

		assert.Len(t, result, 0)
		assert.NotNil(t, result) // Should return empty slice, not nil
	})

	t.Run("Invalid field types", func(t *testing.T) {
		marketsData := []interface{}{
			map[string]interface{}{
				"id":                          []string{"invalid", "id"},    // Wrong type
				"symbol":                      123,                          // Wrong type
				"name":                        true,                         // Wrong type
				"image":                       map[string]string{"url": ""}, // Wrong type
				"current_price":               "not_a_number",               // Wrong type
				"market_cap":                  []float64{123.45},            // Wrong type
				"total_volume":                map[string]interface{}{},     // Wrong type
				"price_change_percentage_24h": "not_a_percentage",           // Wrong type
			},
		}

		result := ConvertMarketsResponseToCoinData(marketsData)

		assert.Len(t, result, 1)

		// All fields should have default values due to type mismatches
		coin := result[0]
		assert.Equal(t, "", coin.ID)
		assert.Equal(t, "", coin.Symbol)
		assert.Equal(t, "", coin.Name)
		assert.Equal(t, "", coin.Image)
		assert.Equal(t, 0.0, coin.CurrentPrice)
		assert.Equal(t, 0.0, coin.MarketCap)
		assert.Equal(t, 0.0, coin.TotalVolume)
		assert.Equal(t, 0.0, coin.PriceChangePercentage24h)
	})

	t.Run("Mixed valid and invalid data", func(t *testing.T) {
		marketsData := []interface{}{
			map[string]interface{}{
				"id":            "bitcoin",
				"symbol":        "btc",
				"name":          "Bitcoin",
				"current_price": 50000.0,
			},
			"invalid_item",
			map[string]interface{}{
				"id":            "ethereum",
				"symbol":        "eth",
				"name":          "Ethereum",
				"current_price": 3000.0,
			},
			123,
			map[string]interface{}{
				"id":            "cardano",
				"symbol":        "ada",
				"name":          "Cardano",
				"current_price": 1.5,
			},
		}

		result := ConvertMarketsResponseToCoinData(marketsData)

		// Only the 3 valid maps should be processed
		assert.Len(t, result, 3)
		assert.Equal(t, "bitcoin", result[0].ID)
		assert.Equal(t, "ethereum", result[1].ID)
		assert.Equal(t, "cardano", result[2].ID)
	})
}
