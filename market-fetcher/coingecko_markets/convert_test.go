package coingecko_markets

import (
	"encoding/json"
	"testing"

	"github.com/status-im/market-proxy/currency_ratios"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eurRatio matches the worked example used across the conversion tests: eur is
// 0.9 usd now and was 0.825 usd 24 hours ago.
var eurRatio = currency_ratios.Ratio{Now: 0.9, H24: 0.825}

// marketsRowJSON is a /coins/markets row as CoinGecko returns it
const marketsRowJSON = `{
  "id": "bitcoin",
  "symbol": "btc",
  "name": "Bitcoin",
  "image": "https://example.com/btc.png",
  "current_price": 100000,
  "market_cap": 2000000000,
  "market_cap_rank": 1,
  "fully_diluted_valuation": 2100000000,
  "total_volume": 50000000,
  "high_24h": 101000,
  "low_24h": 98000,
  "price_change_24h": 5000,
  "price_change_percentage_24h": 10,
  "market_cap_change_24h": 100000000,
  "market_cap_change_percentage_24h": 5,
  "circulating_supply": 19700000,
  "total_supply": 21000000,
  "max_supply": 21000000,
  "ath": 120000,
  "ath_change_percentage": -16.5,
  "ath_date": "2021-11-10T14:24:11.849Z",
  "atl": 67.81,
  "atl_change_percentage": 90774.5,
  "atl_date": "2013-07-06T00:00:00.000Z",
  "roi": null,
  "last_updated": "2026-08-24T12:00:00.000Z",
  "price_change_percentage_1h_in_currency": 2,
  "price_change_percentage_24h_in_currency": 10,
  "price_change_percentage_7d_in_currency": 25,
  "sparkline_in_7d": {"price": [90000, 95000, 100000]}
}`

func decodeRow(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var row map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &row))
	return row
}

func sampleResponse(t *testing.T) interfaces.MarketsResponse {
	t.Helper()
	return interfaces.MarketsResponse([]interface{}{decodeRow(t, marketsRowJSON)})
}

func convertedRow(t *testing.T, conversion MarketsConversion) map[string]interface{} {
	t.Helper()
	converted := ConvertMarketsResponse(sampleResponse(t), conversion)
	require.Len(t, converted, 1)
	row, ok := converted[0].(map[string]interface{})
	require.True(t, ok)
	return row
}

func TestConvertMarketsResponse_SpotScaledFields(t *testing.T) {
	row := convertedRow(t, MarketsConversion{Ratio: eurRatio})

	assert.InDelta(t, 90000.0, row["current_price"], 1e-9)
	assert.InDelta(t, 1800000000.0, row["market_cap"], 1e-3)
	assert.InDelta(t, 1890000000.0, row["fully_diluted_valuation"], 1e-3)
	assert.InDelta(t, 45000000.0, row["total_volume"], 1e-3)
	assert.InDelta(t, 90900.0, row["high_24h"], 1e-6)
	assert.InDelta(t, 88200.0, row["low_24h"], 1e-6)
	assert.InDelta(t, 108000.0, row["ath"], 1e-6)
	assert.InDelta(t, 61.029, row["atl"], 1e-6)
}

// TestConvertMarketsResponse_AbsoluteDeltas pins the honest absolute conversion
// against hand-computed values.
//
//	price 24h ago in usd = 100000 - 5000 = 95000
//	price_change_24h in eur = 100000*0.9 - 95000*0.825 = 90000 - 78375 = 11625
//	market cap 24h ago in usd = 2000000000 - 100000000 = 1900000000
//	market_cap_change_24h in eur = 1800000000 - 1900000000*0.825 = 1800000000 - 1567500000 = 232500000
func TestConvertMarketsResponse_AbsoluteDeltas(t *testing.T) {
	row := convertedRow(t, MarketsConversion{Ratio: eurRatio})

	assert.InDelta(t, 11625.0, row["price_change_24h"], 1e-6)
	assert.InDelta(t, 232500000.0, row["market_cap_change_24h"], 1e-3)
}

// TestConvertMarketsResponse_PercentChanges checks the honest 24h formula:
// ((1 + 0.10) * 0.9 / 0.825 - 1) * 100 = 20
func TestConvertMarketsResponse_PercentChanges(t *testing.T) {
	row := convertedRow(t, MarketsConversion{Ratio: eurRatio})

	assert.InDelta(t, 20.0, row["price_change_percentage_24h"], 1e-9)
	assert.InDelta(t, 20.0, row["price_change_percentage_24h_in_currency"], 1e-9)
	// ((1 + 0.05) * 0.9 / 0.825 - 1) * 100 = 14.5454...
	assert.InDelta(t, (1.05*0.9/0.825-1)*100, row["market_cap_change_percentage_24h"], 1e-9)
}

// TestConvertMarketsResponse_PercentChange1h uses the 1h-ago ratio:
// spot 1h ago 0.88, now 0.9, usd change +2%:
// ((1.02) * 0.9 / 0.88 - 1) * 100 = 4.3181...
func TestConvertMarketsResponse_PercentChange1h(t *testing.T) {
	row := convertedRow(t, MarketsConversion{Ratio: eurRatio, Spot1hAgo: 0.88, Has1hAgo: true})

	assert.InDelta(t, (1.02*0.9/0.88-1)*100, row[PercentChange1hField], 1e-9)
}

// TestConvertMarketsResponse_PercentChange1hLeftAloneWithoutHistory documents the
// fallback: fiat rates move negligibly over an hour, so an unconverted 1h change
// is far better than one computed against the wrong ratio.
func TestConvertMarketsResponse_PercentChange1hLeftAloneWithoutHistory(t *testing.T) {
	row := convertedRow(t, MarketsConversion{Ratio: eurRatio})

	assert.Equal(t, 2.0, row[PercentChange1hField])
}

func TestConvertMarketsResponse_Sparkline(t *testing.T) {
	row := convertedRow(t, MarketsConversion{Ratio: eurRatio})

	sparkline, ok := row["sparkline_in_7d"].(map[string]interface{})
	require.True(t, ok)

	prices, ok := sparkline["price"].([]interface{})
	require.True(t, ok)
	require.Len(t, prices, 3)

	assert.InDelta(t, 81000.0, prices[0], 1e-6)
	assert.InDelta(t, 85500.0, prices[1], 1e-6)
	assert.InDelta(t, 90000.0, prices[2], 1e-6)
}

func TestConvertMarketsResponse_NonMoneyFieldsPassThrough(t *testing.T) {
	original := decodeRow(t, marketsRowJSON)
	row := convertedRow(t, MarketsConversion{Ratio: eurRatio})

	for _, field := range []string{
		"id", "symbol", "name", "image", "market_cap_rank",
		"circulating_supply", "total_supply", "max_supply",
		"ath_date", "atl_date", "roi", "last_updated",
		// documented approximations: honest values need historical fx
		"ath_change_percentage", "atl_change_percentage",
		// windows longer than 24h have no ratio history
		"price_change_percentage_7d_in_currency",
	} {
		assert.Equal(t, original[field], row[field], "field %s must pass through unchanged", field)
	}
}

// TestConvertMarketsResponse_DoesNotMutatePassthrough guards the ADR's core rule
func TestConvertMarketsResponse_DoesNotMutatePassthrough(t *testing.T) {
	response := sampleResponse(t)
	pristine := decodeRow(t, marketsRowJSON)

	converted := ConvertMarketsResponse(response, MarketsConversion{Ratio: eurRatio, Spot1hAgo: 0.88, Has1hAgo: true})

	assert.Equal(t, pristine, response[0], "the source row must be untouched")
	assert.NotEqual(t, response[0], converted[0], "the converted row must be a fresh copy")

	// the nested sparkline must be a fresh structure too
	sourceSparkline := response[0].(map[string]interface{})["sparkline_in_7d"].(map[string]interface{})
	convertedSparkline := converted[0].(map[string]interface{})["sparkline_in_7d"].(map[string]interface{})
	assert.Equal(t, []interface{}{90000.0, 95000.0, 100000.0}, sourceSparkline["price"])
	assert.NotEqual(t, sourceSparkline["price"], convertedSparkline["price"])
}

func TestConvertMarketsResponse_IdentityRatioPassesValuesThrough(t *testing.T) {
	response := sampleResponse(t)

	converted := ConvertMarketsResponse(response, MarketsConversion{
		Ratio:     currency_ratios.IdentityRatio,
		Spot1hAgo: 1,
		Has1hAgo:  true,
	})

	require.Len(t, converted, 1)

	original := response[0].(map[string]interface{})
	result := converted[0].(map[string]interface{})

	for field, value := range original {
		if field == "sparkline_in_7d" {
			continue
		}
		assert.Equal(t, value, result[field], "field %s must be unchanged for the base currency", field)
	}
	assert.Equal(t, original["sparkline_in_7d"], result["sparkline_in_7d"])
}

func TestConvertMarketsResponse_HandlesMissingAndOddRows(t *testing.T) {
	response := interfaces.MarketsResponse([]interface{}{
		map[string]interface{}{"id": "sparse", "current_price": 10.0},
		"not-a-row",
		map[string]interface{}{"id": "nulls", "current_price": nil, "price_change_24h": nil},
	})

	converted := ConvertMarketsResponse(response, MarketsConversion{Ratio: eurRatio})
	require.Len(t, converted, 3)

	sparse := converted[0].(map[string]interface{})
	assert.InDelta(t, 9.0, sparse["current_price"], 1e-9)
	assert.NotContains(t, sparse, "market_cap")

	assert.Equal(t, "not-a-row", converted[1])

	nulls := converted[2].(map[string]interface{})
	assert.Nil(t, nulls["current_price"])
	assert.Nil(t, nulls["price_change_24h"])
}

func TestConvertMarketsResponse_Empty(t *testing.T) {
	converted := ConvertMarketsResponse(interfaces.MarketsResponse{}, MarketsConversion{Ratio: eurRatio})
	require.NotNil(t, converted)
	assert.Empty(t, converted)
}

// TestConvertMarketsResponse_BitcoinInBtcTerms is the case a spot-only conversion
// gets wrong: bitcoin's own 24h change measured in btc must be ~0.
func TestConvertMarketsResponse_BitcoinInBtcTerms(t *testing.T) {
	btcRatio := currency_ratios.Ratio{Now: 1e-5, H24: 1.1e-5}

	converted := ConvertMarketsResponse(sampleResponse(t), MarketsConversion{Ratio: btcRatio})
	row := converted[0].(map[string]interface{})

	assert.InDelta(t, 1.0, row["current_price"], 1e-9)
	assert.InDelta(t, 0.0, row["price_change_percentage_24h"], 1e-9)
	// absolute delta: 100000*1e-5 - 95000*1.1e-5 = 1 - 1.045 = -0.045
	assert.InDelta(t, -0.045, row["price_change_24h"], 1e-9)
}
