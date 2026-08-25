package coingecko_leaderboard

import (
	"encoding/json"
	"testing"

	"github.com/status-im/market-proxy/currency_ratios"
	cg "github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertQuotes_MissingPercentChangeIsNotFabricated is the bug this file
// exists for.
//
// The upstream row carries no usd_24h_change - the proxy only asks for it when
// include_24hr_change is set, and CoinGecko omits it for tokens too new to have
// one. Collapsing that to 0.0 used to look harmless, because a "0% change"
// reads like a plausible default. It is not harmless: the honest conversion of
// 0% is not 0%, it is whatever the fx ratio moved by over the window. The
// client would have been handed a real-looking percentage the provider never
// reported, and only for converted currencies - the base currency short-circuit
// hides it, which is why it survived review.
func TestConvertQuotes_MissingPercentChangeIsNotFabricated(t *testing.T) {
	// A ratio that moved: eur strengthened against usd over the window
	movedRatio := currency_ratios.Ratio{Now: 0.9, H24: 0.825}

	// Sanity check that this ratio does fabricate a number from a zero
	fabricated := currency_ratios.ConvertPercentChange24h(0, movedRatio)
	require.NotZero(t, fabricated, "the ratio must have moved for this test to mean anything")
	require.InDelta(t, (0.9/0.825-1)*100, fabricated, 1e-9)

	quotes := ConvertPriceResponseToPriceQuotes(cg.SimplePriceResponse{
		"newcoin": map[string]interface{}{"usd": 12.5},
	}, "usd")

	require.Contains(t, quotes, "newcoin")
	require.Nil(t, quotes["newcoin"].PercentChange24h, "an unreported change must stay unreported")

	converted := ConvertQuotes(quotes, movedRatio)

	require.Contains(t, converted, "newcoin")
	assert.Nil(t, converted["newcoin"].PercentChange24h,
		"converting an absent change must not invent %v%%", fabricated)
	assert.InDelta(t, 12.5*0.9, converted["newcoin"].Price, 1e-9, "the price still converts")
}

// TestConvertAPIResponse_MissingPercentChangeIsNotFabricated is the same case on
// the markets rows
func TestConvertAPIResponse_MissingPercentChangeIsNotFabricated(t *testing.T) {
	movedRatio := currency_ratios.Ratio{Now: 0.9, H24: 0.825}

	rows := ConvertMarketsResponseToCoinData([]interface{}{
		map[string]interface{}{
			"id":            "newcoin",
			"symbol":        "new",
			"current_price": 12.5,
			// no market_cap, no total_volume, no price_change_percentage_24h
		},
	})
	require.Len(t, rows, 1)
	require.Nil(t, rows[0].PriceChangePercentage24h)

	converted := ConvertAPIResponse(&APIResponse{Data: rows}, movedRatio)
	require.NotNil(t, converted)
	require.Len(t, converted.Data, 1)

	row := converted.Data[0]
	assert.Nil(t, row.PriceChangePercentage24h, "an absent change must not become a real-looking one")
	assert.Nil(t, row.MarketCap, "an absent market cap must not become 0")
	assert.Nil(t, row.TotalVolume, "an absent volume must not become 0")
	assert.InDelta(t, 12.5*0.9, requireFloat(t, row.CurrentPrice), 1e-9)
}

// TestConvert_RealZeroSurvives is the other half of the contract: a value the
// provider genuinely reported as zero is data, and must not be dropped.
func TestConvert_RealZeroSurvives(t *testing.T) {
	movedRatio := currency_ratios.Ratio{Now: 0.9, H24: 0.825}

	quotes := ConvertPriceResponseToPriceQuotes(cg.SimplePriceResponse{
		"stable": map[string]interface{}{
			"usd":            1.0,
			"usd_24h_change": 0.0,
			"usd_24h_vol":    0.0,
		},
	}, "usd")

	require.NotNil(t, quotes["stable"].PercentChange24h)
	assert.Equal(t, 0.0, *quotes["stable"].PercentChange24h)
	require.NotNil(t, quotes["stable"].Volume24h)

	converted := ConvertQuotes(quotes, movedRatio)

	// A flat usd price is genuinely not flat in a currency that moved - this is
	// the honest conversion doing its job on a value that really was zero.
	require.NotNil(t, converted["stable"].PercentChange24h)
	assert.InDelta(t, (0.9/0.825-1)*100, *converted["stable"].PercentChange24h, 1e-9)
	require.NotNil(t, converted["stable"].Volume24h)
	assert.Equal(t, 0.0, *converted["stable"].Volume24h)
}

// TestLeaderboardJSON_AbsentFieldsAreOmitted pins the wire format. Clients such
// as status-go decode into plain float64s, so an omitted key decodes as 0 there
// - that is the intended "not reported" reading, and omitting keys must not
// break decoding.
func TestLeaderboardJSON_AbsentFieldsAreOmitted(t *testing.T) {
	response := &APIResponse{Data: []CoinData{
		{ID: "sparse", Symbol: "sps", Name: "Sparse", CurrentPrice: floatPtr(1.5)},
		{
			ID:                       "full",
			Symbol:                   "fll",
			Name:                     "Full",
			Image:                    "https://example.com/f.png",
			CurrentPrice:             floatPtr(2),
			MarketCap:                floatPtr(3),
			TotalVolume:              floatPtr(4),
			PriceChangePercentage24h: floatPtr(0),
		},
	}}

	encoded, err := json.Marshal(response)
	require.NoError(t, err)

	assert.JSONEq(t, `{"data":[
		{"id":"sparse","symbol":"sps","name":"Sparse","image":"","current_price":1.5},
		{"id":"full","symbol":"fll","name":"Full","image":"https://example.com/f.png",
		 "current_price":2,"market_cap":3,"total_volume":4,"price_change_percentage_24h":0}
	]}`, string(encoded))

	// A client decoding into plain floats still works; absent reads as zero
	var decoded struct {
		Data []struct {
			ID                       string  `json:"id"`
			CurrentPrice             float64 `json:"current_price"`
			MarketCap                float64 `json:"market_cap"`
			PriceChangePercentage24h float64 `json:"price_change_percentage_24h"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Len(t, decoded.Data, 2)
	assert.Equal(t, 1.5, decoded.Data[0].CurrentPrice)
	assert.Equal(t, 0.0, decoded.Data[0].MarketCap, "an omitted field decodes as zero, as intended")
	assert.Equal(t, 0.0, decoded.Data[1].PriceChangePercentage24h, "a reported zero decodes as zero too")
}

func TestQuoteJSON_AbsentFieldsAreOmitted(t *testing.T) {
	encoded, err := json.Marshal(PriceQuotes{
		"sparse": {Price: 1.5},
		"full":   {Price: 2, Volume24h: floatPtr(3), MarketCap: floatPtr(4), PercentChange24h: floatPtr(0)},
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"sparse": {"price":1.5},
		"full": {"price":2,"volume_24h":3,"market_cap":4,"percent_change_24h":0}
	}`, string(encoded))

	var decoded map[string]struct {
		Price            float64 `json:"price"`
		PercentChange24h float64 `json:"percent_change_24h"`
	}
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, 1.5, decoded["sparse"].Price)
	assert.Equal(t, 0.0, decoded["sparse"].PercentChange24h)
}

// TestConvert_DoesNotAliasSourcePointers guards the copy: the converted rows
// must own their values, not write through the pointers the cache holds.
func TestConvert_DoesNotAliasSourcePointers(t *testing.T) {
	source := &APIResponse{Data: []CoinData{{
		ID:                       "bitcoin",
		CurrentPrice:             floatPtr(100000),
		PriceChangePercentage24h: floatPtr(10),
	}}}

	converted := ConvertAPIResponse(source, currency_ratios.Ratio{Now: 0.9, H24: 0.825})

	assert.Equal(t, 100000.0, *source.Data[0].CurrentPrice, "the cached value is untouched")
	assert.Equal(t, 10.0, *source.Data[0].PriceChangePercentage24h)
	assert.NotSame(t, source.Data[0].CurrentPrice, converted.Data[0].CurrentPrice)
	assert.NotSame(t, source.Data[0].PriceChangePercentage24h, converted.Data[0].PriceChangePercentage24h)

	quotes := PriceQuotes{"bitcoin": {Price: 100000, MarketCap: floatPtr(2e9), PercentChange24h: floatPtr(10)}}
	convertedQuotes := ConvertQuotes(quotes, currency_ratios.Ratio{Now: 0.9, H24: 0.825})

	assert.Equal(t, 2e9, *quotes["bitcoin"].MarketCap)
	assert.NotSame(t, quotes["bitcoin"].MarketCap, convertedQuotes["bitcoin"].MarketCap)
}
