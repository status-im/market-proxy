package coingecko_leaderboard

import (
	"testing"

	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eurRatio matches the worked example in currency_ratios: eur is 0.9 usd now
// and was 0.825 usd 24 hours ago.
var eurRatio = currency_ratios.Ratio{Now: 0.9, H24: 0.825}

func sampleAPIResponse() *APIResponse {
	return &APIResponse{
		Data: []CoinData{
			{
				ID:                       "bitcoin",
				Symbol:                   "btc",
				Name:                     "Bitcoin",
				Image:                    "https://example.com/btc.png",
				CurrentPrice:             100000,
				MarketCap:                2000000000,
				TotalVolume:              50000000,
				PriceChangePercentage24h: 10,
			},
			{
				ID:                       "ethereum",
				Symbol:                   "eth",
				Name:                     "Ethereum",
				CurrentPrice:             4000,
				MarketCap:                480000000,
				TotalVolume:              20000000,
				PriceChangePercentage24h: -5,
			},
		},
	}
}

func TestConvertAPIResponse(t *testing.T) {
	original := sampleAPIResponse()
	converted := ConvertAPIResponse(original, eurRatio)

	require.NotNil(t, converted)
	require.Len(t, converted.Data, 2)

	btc := converted.Data[0]
	assert.Equal(t, "bitcoin", btc.ID)
	assert.Equal(t, "btc", btc.Symbol)
	assert.Equal(t, "Bitcoin", btc.Name)
	assert.Equal(t, "https://example.com/btc.png", btc.Image)
	assert.InDelta(t, 90000.0, btc.CurrentPrice, 1e-9)
	assert.InDelta(t, 1800000000.0, btc.MarketCap, 1e-3)
	assert.InDelta(t, 45000000.0, btc.TotalVolume, 1e-3)
	// ((1 + 0.10) * 0.9 / 0.825 - 1) * 100 = 20
	assert.InDelta(t, 20.0, btc.PriceChangePercentage24h, 1e-9)

	eth := converted.Data[1]
	assert.InDelta(t, 3600.0, eth.CurrentPrice, 1e-9)
	// ((1 - 0.05) * 0.9 / 0.825 - 1) * 100 = 3.6363...
	assert.InDelta(t, (0.95*0.9/0.825-1)*100, eth.PriceChangePercentage24h, 1e-9)
}

// TestConvertAPIResponse_DoesNotMutatePassthrough guards the ADR's core rule:
// cached Passthrough data is never mutated by a conversion request.
func TestConvertAPIResponse_DoesNotMutatePassthrough(t *testing.T) {
	original := sampleAPIResponse()
	pristine := sampleAPIResponse()

	converted := ConvertAPIResponse(original, eurRatio)

	assert.Equal(t, pristine, original)
	assert.NotSame(t, original, converted)
	assert.NotEqual(t, original.Data[0].CurrentPrice, converted.Data[0].CurrentPrice)
}

func TestConvertAPIResponse_IdentityRatioPassesValuesThrough(t *testing.T) {
	original := sampleAPIResponse()
	converted := ConvertAPIResponse(original, currency_ratios.IdentityRatio)

	assert.Equal(t, original.Data, converted.Data)
}

func TestConvertAPIResponse_NilAndEmpty(t *testing.T) {
	assert.Nil(t, ConvertAPIResponse(nil, eurRatio))

	converted := ConvertAPIResponse(&APIResponse{Data: []CoinData{}}, eurRatio)
	require.NotNil(t, converted)
	assert.Empty(t, converted.Data)
}

func sampleQuotes() map[string]Quote {
	return map[string]Quote{
		"bitcoin": {Price: 100000, Volume24h: 50000000, MarketCap: 2000000000, PercentChange24h: 10},
		"tether":  {Price: 1, Volume24h: 1000000, MarketCap: 100000000, PercentChange24h: 0},
	}
}

func TestConvertQuotes(t *testing.T) {
	original := sampleQuotes()
	converted := ConvertQuotes(original, eurRatio)

	require.Len(t, converted, 2)

	btc := converted["bitcoin"]
	assert.InDelta(t, 90000.0, btc.Price, 1e-9)
	assert.InDelta(t, 45000000.0, btc.Volume24h, 1e-3)
	assert.InDelta(t, 1800000000.0, btc.MarketCap, 1e-3)
	assert.InDelta(t, 20.0, btc.PercentChange24h, 1e-9)

	// A stablecoin flat in usd is not flat in eur when eur moved
	usdt := converted["tether"]
	assert.InDelta(t, 0.9, usdt.Price, 1e-9)
	assert.InDelta(t, (0.9/0.825-1)*100, usdt.PercentChange24h, 1e-9)
}

func TestConvertQuotes_DoesNotMutatePassthrough(t *testing.T) {
	original := sampleQuotes()
	pristine := sampleQuotes()

	ConvertQuotes(original, eurRatio)

	assert.Equal(t, pristine, original)
}

func TestConvertQuotes_Empty(t *testing.T) {
	converted := ConvertQuotes(map[string]Quote{}, eurRatio)
	require.NotNil(t, converted)
	assert.Empty(t, converted)

	converted = ConvertQuotes(nil, eurRatio)
	require.NotNil(t, converted)
	assert.Empty(t, converted)
}

// TestConvertQuotes_BitcoinInBtcTerms is the case a spot-only conversion gets
// wrong: bitcoin's 24h change measured in btc must be ~0.
func TestConvertQuotes_BitcoinInBtcTerms(t *testing.T) {
	btcRatio := currency_ratios.Ratio{Now: 1e-5, H24: 1.1e-5}

	converted := ConvertQuotes(map[string]Quote{
		"bitcoin": {Price: 100000, PercentChange24h: 10},
	}, btcRatio)

	assert.InDelta(t, 1.0, converted["bitcoin"].Price, 1e-9)
	assert.InDelta(t, 0.0, converted["bitcoin"].PercentChange24h, 1e-9)
}
