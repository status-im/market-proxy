package coingecko_prices

import (
	"testing"

	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eurRatio: eur is 0.9 usd now and was 0.825 usd 24 hours ago
var eurRatio = currency_ratios.Ratio{Now: 0.9, H24: 0.825}

func sampleSimplePrices() interfaces.SimplePriceResponse {
	return interfaces.SimplePriceResponse{
		"bitcoin": map[string]interface{}{
			"usd":             100000.0,
			"usd_market_cap":  2000000000.0,
			"usd_24h_vol":     50000000.0,
			"usd_24h_change":  10.0,
			"last_updated_at": 1703097600.0,
		},
	}
}

func convertedRow(t *testing.T, response interfaces.SimplePriceResponse, currency string, ratio currency_ratios.Ratio, keepBase bool) map[string]interface{} {
	t.Helper()

	converted := ConvertSimplePrices(response, currency, ratio, "", keepBase)
	row, ok := converted["bitcoin"].(map[string]interface{})
	require.True(t, ok)

	return row
}

func TestConvertSimplePrices_AddsEstimateKeys(t *testing.T) {
	row := convertedRow(t, sampleSimplePrices(), "eur", eurRatio, false)

	assert.InDelta(t, 90000.0, row["eur"], 1e-9)
	assert.InDelta(t, 1800000000.0, row["eur_market_cap"], 1e-3)
	assert.InDelta(t, 45000000.0, row["eur_24h_vol"], 1e-3)
	// ((1 + 0.10) * 0.9 / 0.825 - 1) * 100 = 20
	assert.InDelta(t, 20.0, row["eur_24h_change"], 1e-9)

	// last_updated_at is not a currency field and passes through
	assert.Equal(t, 1703097600.0, row["last_updated_at"])
}

// TestConvertSimplePrices_DropsBaseKeysWhenNotRequested covers
// vs_currencies=chf&convert_currency=eur: usd is only read in to compute the
// Estimate and must not leak into the response.
func TestConvertSimplePrices_DropsBaseKeysWhenNotRequested(t *testing.T) {
	response := sampleSimplePrices()
	response["bitcoin"].(map[string]interface{})["chf"] = 88000.0

	row := convertedRow(t, response, "eur", eurRatio, false)

	assert.NotContains(t, row, "usd")
	assert.NotContains(t, row, "usd_market_cap")
	assert.NotContains(t, row, "usd_24h_vol")
	assert.NotContains(t, row, "usd_24h_change")

	assert.Equal(t, 88000.0, row["chf"], "other passthrough currencies are untouched")
	assert.Contains(t, row, "eur")
}

// TestConvertSimplePrices_KeepsBaseKeysWhenRequested covers
// vs_currencies=usd&convert_currency=eur: both key sets are returned.
func TestConvertSimplePrices_KeepsBaseKeysWhenRequested(t *testing.T) {
	row := convertedRow(t, sampleSimplePrices(), "eur", eurRatio, true)

	assert.Equal(t, 100000.0, row["usd"])
	assert.Equal(t, 2000000000.0, row["usd_market_cap"])
	assert.Equal(t, 50000000.0, row["usd_24h_vol"])
	assert.Equal(t, 10.0, row["usd_24h_change"])

	assert.InDelta(t, 90000.0, row["eur"], 1e-9)
	assert.InDelta(t, 20.0, row["eur_24h_change"], 1e-9)
}

// TestConvertSimplePrices_OnlyAddsKeysWithASource mirrors the include_* flags:
// a key the caller filtered out has no base value to convert from.
func TestConvertSimplePrices_OnlyAddsKeysWithASource(t *testing.T) {
	response := interfaces.SimplePriceResponse{
		"bitcoin": map[string]interface{}{
			"usd": 100000.0,
		},
	}

	row := convertedRow(t, response, "eur", eurRatio, false)

	assert.InDelta(t, 90000.0, row["eur"], 1e-9)
	assert.NotContains(t, row, "eur_market_cap")
	assert.NotContains(t, row, "eur_24h_vol")
	assert.NotContains(t, row, "eur_24h_change")
}

func TestConvertSimplePrices_BaseCurrencyPassesValuesThrough(t *testing.T) {
	response := sampleSimplePrices()
	response["bitcoin"].(map[string]interface{})["eur"] = 88000.0

	// vs_currencies=eur&convert_currency=usd - usd is not requested as
	// Passthrough, so keepBase is false and the usd keys are re-created as Estimates
	row := convertedRow(t, response, "usd", currency_ratios.IdentityRatio, false)

	assert.Equal(t, 100000.0, row["usd"])
	assert.Equal(t, 2000000000.0, row["usd_market_cap"])
	assert.Equal(t, 50000000.0, row["usd_24h_vol"])
	assert.Equal(t, 10.0, row["usd_24h_change"])
	assert.Equal(t, 88000.0, row["eur"])
}

func TestConvertSimplePrices_DoesNotMutatePassthrough(t *testing.T) {
	response := sampleSimplePrices()
	pristine := sampleSimplePrices()

	ConvertSimplePrices(response, "eur", eurRatio, "", false)

	assert.Equal(t, pristine, response)
}

func TestConvertSimplePrices_AppliesPrecision(t *testing.T) {
	converted := ConvertSimplePrices(sampleSimplePrices(), "eur", eurRatio, "2", false)
	row := converted["bitcoin"].(map[string]interface{})

	assert.Equal(t, 20.0, row["eur_24h_change"])
	assert.Equal(t, 90000.0, row["eur"])

	// a value with more decimals gets rounded like Passthrough values do
	response := interfaces.SimplePriceResponse{
		"tiny": map[string]interface{}{"usd": 0.123456789},
	}
	converted = ConvertSimplePrices(response, "eur", eurRatio, "3", false)
	assert.Equal(t, 0.111, converted["tiny"].(map[string]interface{})["eur"])
}

func TestConvertSimplePrices_HandlesOddRows(t *testing.T) {
	response := interfaces.SimplePriceResponse{
		"weird":   "not-a-row",
		"nulls":   map[string]interface{}{"usd": nil, "last_updated_at": 1.0},
		"nousd":   map[string]interface{}{"chf": 5.0},
		"bitcoin": sampleSimplePrices()["bitcoin"],
	}

	converted := ConvertSimplePrices(response, "eur", eurRatio, "", false)

	assert.Equal(t, "not-a-row", converted["weird"])
	assert.NotContains(t, converted["nulls"].(map[string]interface{}), "eur")
	assert.NotContains(t, converted["nousd"].(map[string]interface{}), "eur")
	assert.Equal(t, 5.0, converted["nousd"].(map[string]interface{})["chf"])
	assert.Contains(t, converted["bitcoin"].(map[string]interface{}), "eur")
}

func TestSourceCurrencies(t *testing.T) {
	tests := []struct {
		name      string
		requested []string
		expected  []string
	}{
		{name: "adds the base currency", requested: []string{"chf"}, expected: []string{"chf", "usd"}},
		{name: "keeps the list when base is present", requested: []string{"usd", "chf"}, expected: []string{"usd", "chf"}},
		{name: "matches base case-insensitively", requested: []string{"USD"}, expected: []string{"USD"}},
		{name: "handles an empty list", requested: []string{}, expected: []string{"usd"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, SourceCurrencies(tt.requested))
		})
	}
}

func TestContainsCurrency(t *testing.T) {
	assert.True(t, ContainsCurrency([]string{"usd", "eur"}, "eur"))
	assert.True(t, ContainsCurrency([]string{"USD"}, "usd"))
	assert.False(t, ContainsCurrency([]string{"usd", "eur"}, "chf"))
	assert.False(t, ContainsCurrency(nil, "usd"))
}

func TestIsCurrencyField(t *testing.T) {
	assert.True(t, isCurrencyField("usd", "usd"))
	assert.True(t, isCurrencyField("usd_market_cap", "usd"))
	assert.True(t, isCurrencyField("usd_24h_vol", "usd"))
	assert.True(t, isCurrencyField("usd_24h_change", "usd"))
	assert.False(t, isCurrencyField("last_updated_at", "usd"))
	assert.False(t, isCurrencyField("eur", "usd"))
	assert.False(t, isCurrencyField("eur_market_cap", "usd"))
}
