package coingecko_prices

import (
	cg "github.com/status-im/market-proxy/coingecko_common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripResponse(t *testing.T) {
	// Create test response with all possible fields
	testResponse := cg.SimplePriceResponse{
		"bitcoin": map[string]interface{}{
			"usd":             50000.0,
			"eur":             42000.0,
			"btc":             1.0,
			"usd_market_cap":  950000000000.0,
			"eur_market_cap":  798000000000.0,
			"btc_market_cap":  19000000.0,
			"usd_24h_vol":     25000000000.0,
			"eur_24h_vol":     21000000000.0,
			"btc_24h_vol":     500000.0,
			"usd_24h_change":  2.5,
			"eur_24h_change":  1.8,
			"btc_24h_change":  0.0,
			"last_updated_at": 1749059921,
		},
		"ethereum": map[string]interface{}{
			"usd":             3000.0,
			"eur":             2520.0,
			"usd_market_cap":  360000000000.0,
			"eur_market_cap":  302400000000.0,
			"usd_24h_vol":     15000000000.0,
			"eur_24h_vol":     12600000000.0,
			"usd_24h_change":  -1.2,
			"eur_24h_change":  -1.5,
			"last_updated_at": 1749059921,
		},
	}

	// Test 1: Only basic currencies
	params1 := cg.PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd", "eur"},
	}
	result1 := stripResponse(testResponse, params1)

	assert.Len(t, result1, 2)
	assert.Contains(t, result1, "bitcoin")
	assert.Contains(t, result1, "ethereum")

	bitcoinData1 := result1["bitcoin"].(map[string]interface{})
	assert.Contains(t, bitcoinData1, "usd")
	assert.Contains(t, bitcoinData1, "eur")
	assert.NotContains(t, bitcoinData1, "btc")
	assert.NotContains(t, bitcoinData1, "usd_market_cap")
	assert.NotContains(t, bitcoinData1, "last_updated_at")

	// Test 2: With market cap
	params2 := cg.PriceParams{
		IDs:              []string{"bitcoin"},
		Currencies:       []string{"usd"},
		IncludeMarketCap: true,
	}
	result2 := stripResponse(testResponse, params2)

	bitcoinData2 := result2["bitcoin"].(map[string]interface{})
	assert.Contains(t, bitcoinData2, "usd")
	assert.Contains(t, bitcoinData2, "usd_market_cap")
	assert.NotContains(t, bitcoinData2, "eur_market_cap") // not requested currency
	assert.NotContains(t, bitcoinData2, "usd_24h_vol")

	// Test 3: With all optional fields
	params3 := cg.PriceParams{
		IDs:                  []string{"bitcoin"},
		Currencies:           []string{"usd", "btc"},
		IncludeMarketCap:     true,
		Include24hrVol:       true,
		Include24hrChange:    true,
		IncludeLastUpdatedAt: true,
	}
	result3 := stripResponse(testResponse, params3)

	bitcoinData3 := result3["bitcoin"].(map[string]interface{})
	assert.Contains(t, bitcoinData3, "usd")
	assert.Contains(t, bitcoinData3, "btc")
	assert.Contains(t, bitcoinData3, "usd_market_cap")
	assert.Contains(t, bitcoinData3, "btc_market_cap")
	assert.Contains(t, bitcoinData3, "usd_24h_vol")
	assert.Contains(t, bitcoinData3, "btc_24h_vol")
	assert.Contains(t, bitcoinData3, "usd_24h_change")
	assert.Contains(t, bitcoinData3, "btc_24h_change")
	assert.Contains(t, bitcoinData3, "last_updated_at")
	assert.NotContains(t, bitcoinData3, "eur") // not requested
}

func TestStripResponseWithPrecision(t *testing.T) {
	testResponse := cg.SimplePriceResponse{
		"bitcoin": map[string]interface{}{
			"usd":             50123.456789,
			"usd_market_cap":  950123456789.123,
			"usd_24h_change":  2.123456789,
			"last_updated_at": 1749059921, // should not be rounded
		},
	}

	params := cg.PriceParams{
		IDs:                  []string{"bitcoin"},
		Currencies:           []string{"usd"},
		IncludeMarketCap:     true,
		Include24hrChange:    true,
		IncludeLastUpdatedAt: true,
		Precision:            "2", // 2 decimal places
	}

	result := stripResponse(testResponse, params)
	bitcoinData := result["bitcoin"].(map[string]interface{})

	assert.Equal(t, 50123.46, bitcoinData["usd"])
	assert.Equal(t, 950123456789.12, bitcoinData["usd_market_cap"])
	assert.Equal(t, 2.12, bitcoinData["usd_24h_change"])
	assert.Equal(t, 1749059921, bitcoinData["last_updated_at"]) // timestamp not rounded
}

func TestShouldIncludeField(t *testing.T) {
	requestedCurrencies := map[string]bool{
		"usd": true,
		"eur": true,
	}

	params := cg.PriceParams{
		Currencies:           []string{"usd", "eur"},
		IncludeMarketCap:     true,
		Include24hrVol:       false,
		Include24hrChange:    true,
		IncludeLastUpdatedAt: true,
	}

	// Test base currencies
	assert.True(t, shouldIncludeField("usd", requestedCurrencies, params))
	assert.True(t, shouldIncludeField("eur", requestedCurrencies, params))
	assert.False(t, shouldIncludeField("btc", requestedCurrencies, params))

	// Test market cap fields
	assert.True(t, shouldIncludeField("usd_market_cap", requestedCurrencies, params))
	assert.True(t, shouldIncludeField("eur_market_cap", requestedCurrencies, params))
	assert.False(t, shouldIncludeField("btc_market_cap", requestedCurrencies, params))

	// Test volume fields (disabled)
	assert.False(t, shouldIncludeField("usd_24h_vol", requestedCurrencies, params))
	assert.False(t, shouldIncludeField("eur_24h_vol", requestedCurrencies, params))

	// Test change fields
	assert.True(t, shouldIncludeField("usd_24h_change", requestedCurrencies, params))
	assert.True(t, shouldIncludeField("eur_24h_change", requestedCurrencies, params))
	assert.False(t, shouldIncludeField("btc_24h_change", requestedCurrencies, params))

	// Test last updated field
	assert.True(t, shouldIncludeField("last_updated_at", requestedCurrencies, params))
}

func TestRoundToPrecision(t *testing.T) {
	// Test with different precision values
	assert.Equal(t, 123.46, roundToPrecision(123.456789, 2))
	assert.Equal(t, 123.5, roundToPrecision(123.456789, 1))
	assert.Equal(t, 123.0, roundToPrecision(123.456789, 0))       // round to integer
	assert.Equal(t, 123.456789, roundToPrecision(123.456789, -1)) // negative precision returns original

	// Test rounding edge cases
	assert.Equal(t, 123.46, roundToPrecision(123.455, 2)) // round down
	assert.Equal(t, 123.46, roundToPrecision(123.456, 2)) // round up

	// Test with zero
	assert.Equal(t, 0.0, roundToPrecision(0.0, 2))

	// Test rounding to integer
	assert.Equal(t, 124.0, roundToPrecision(123.6, 0))
	assert.Equal(t, 123.0, roundToPrecision(123.4, 0))
}

func TestIsNumericField(t *testing.T) {
	// Numeric fields that should be rounded
	assert.True(t, isNumericField("usd"))
	assert.True(t, isNumericField("eur"))
	assert.True(t, isNumericField("usd_market_cap"))
	assert.True(t, isNumericField("usd_24h_vol"))
	assert.True(t, isNumericField("usd_24h_change"))

	// Timestamp field that should not be rounded
	assert.False(t, isNumericField("last_updated_at"))
}
