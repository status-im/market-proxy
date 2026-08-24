package e2etest

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getJSON performs a GET and decodes the JSON body
func getJSON(t *testing.T, url string, out interface{}) *http.Response {
	t.Helper()

	resp, err := http.Get(url)
	require.NoError(t, err, "request to %s should succeed", url)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	if out != nil && resp.StatusCode == http.StatusOK {
		require.NoError(t, json.Unmarshal(body, out), "response should be valid JSON: %s", string(body))
	}
	if out != nil && resp.StatusCode != http.StatusOK {
		_ = json.Unmarshal(body, out)
	}

	return resp
}

// waitForExchangeRates waits until the exchange rates snapshot is available
func waitForExchangeRates(t *testing.T, env *TestEnv) {
	t.Helper()

	require.Eventually(t, func() bool {
		resp, err := http.Get(env.ServerBaseURL + "/api/v1/exchange_rates")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 15*time.Second, 200*time.Millisecond, "exchange rates should become available")
}

// waitForConvertedMarkets waits until the ratios snapshot is populated, i.e. the
// converted markets response stops being empty
func waitForConvertedMarkets(t *testing.T, env *TestEnv) []interface{} {
	t.Helper()

	var data []interface{}
	require.Eventually(t, func() bool {
		var response map[string]interface{}
		resp := getJSON(t, env.ServerBaseURL+"/api/v1/leaderboard/markets?convert_currency=eur", &response)
		if resp.StatusCode != http.StatusOK {
			return false
		}
		items, ok := response["data"].([]interface{})
		if !ok || len(items) == 0 {
			return false
		}
		data = items
		return true
	}, 15*time.Second, 200*time.Millisecond, "converted markets should become available")

	return data
}

// waitForSimplePrices waits until the simple/price cache holds the given query's
// tokens. The prices cache is refilled on every update cycle and can be
// momentarily empty.
func waitForSimplePrices(t *testing.T, env *TestEnv, query string) {
	t.Helper()

	require.Eventually(t, func() bool {
		var response map[string]map[string]float64
		resp := getJSON(t, env.ServerBaseURL+query, &response)
		return resp.StatusCode == http.StatusOK && len(response) > 0
	}, 15*time.Second, 200*time.Millisecond, "simple/price should return data")
}

// TestExchangeRatesEndpoint checks the Passthrough exchange rates endpoint
func TestExchangeRatesEndpoint(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForExchangeRates(t, env)

	resp, err := http.Get(env.ServerBaseURL + "/api/v1/exchange_rates")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("ETag"), "ETag should be set for polling clients")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &response))

	rates, ok := response["rates"].(map[string]interface{})
	require.True(t, ok, "response should contain a 'rates' object")
	assert.Contains(t, rates, "btc")
	assert.Contains(t, rates, "eur")

	// Passthrough: values are served exactly as the provider returned them,
	// full precision included.
	assert.JSONEq(t, defaultExchangeRatesData(), string(body))
}

// TestLeaderboardMarketsConvertCurrency checks the realtime Estimate on
// /api/v1/leaderboard/markets
func TestLeaderboardMarketsConvertCurrency(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForDataInitialization(t, env)

	convertedData := waitForConvertedMarkets(t, env)

	// Passthrough response, for comparison
	var usdResponse map[string]interface{}
	resp := getJSON(t, env.ServerBaseURL+"/api/v1/leaderboard/markets", &usdResponse)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	usdData, ok := usdResponse["data"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, usdData)
	require.Len(t, convertedData, len(usdData), "conversion must not change the number of rows")

	usdCoin := usdData[0].(map[string]interface{})
	eurCoin := convertedData[0].(map[string]interface{})

	assert.Equal(t, usdCoin["id"], eurCoin["id"])
	assert.Equal(t, usdCoin["symbol"], eurCoin["symbol"])

	usdPrice := usdCoin["current_price"].(float64)
	eurPrice := eurCoin["current_price"].(float64)
	assert.InDelta(t, usdPrice*RatioFixtureEURNow, eurPrice, 1e-6)

	usdMarketCap := usdCoin["market_cap"].(float64)
	assert.InDelta(t, usdMarketCap*RatioFixtureEURNow, eurCoin["market_cap"].(float64), 1e-3)

	usdVolume := usdCoin["total_volume"].(float64)
	assert.InDelta(t, usdVolume*RatioFixtureEURNow, eurCoin["total_volume"].(float64), 1e-3)

	// Honest percent change conversion
	usdPct := usdCoin["price_change_percentage_24h"].(float64)
	expectedPct := ((1+usdPct/100)*RatioFixtureEURNow/RatioFixtureEUR24h - 1) * 100
	assert.InDelta(t, expectedPct, eurCoin["price_change_percentage_24h"].(float64), 1e-9)

	// The cached Passthrough response is untouched by the conversion request
	var usdAfter map[string]interface{}
	resp = getJSON(t, env.ServerBaseURL+"/api/v1/leaderboard/markets", &usdAfter)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, usdResponse, usdAfter)
}

// TestLeaderboardMarketsConvertCurrencyToBase checks that convert_currency=usd
// passes the values through numerically
func TestLeaderboardMarketsConvertCurrencyToBase(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForDataInitialization(t, env)
	waitForConvertedMarkets(t, env)

	var usdResponse, convertedResponse map[string]interface{}
	getJSON(t, env.ServerBaseURL+"/api/v1/leaderboard/markets", &usdResponse)
	getJSON(t, env.ServerBaseURL+"/api/v1/leaderboard/markets?convert_currency=usd", &convertedResponse)

	assert.Equal(t, usdResponse, convertedResponse)
}

// TestLeaderboardConvertCurrencyUnknown checks the 400 on both endpoints
func TestLeaderboardConvertCurrencyUnknown(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForDataInitialization(t, env)

	for _, path := range []string{
		"/api/v1/leaderboard/markets?convert_currency=xyz",
		"/api/v1/leaderboard/prices?convert_currency=xyz",
	} {
		var body map[string]string
		resp := getJSON(t, env.ServerBaseURL+path, &body)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "%s should be rejected", path)
		assert.Equal(t, "unsupported convert_currency: xyz", body["error"])
	}
}

// TestCoinsMarketsConvertCurrency checks the realtime Estimate on the
// CoinGecko-compatible /api/v1/coins/markets endpoint
func TestCoinsMarketsConvertCurrency(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForDataInitialization(t, env)
	waitForConvertedMarkets(t, env)

	const path = "/api/v1/coins/markets?ids=bitcoin,ethereum"

	var usdRows []map[string]interface{}
	require.Eventually(t, func() bool {
		usdRows = nil
		resp := getJSON(t, env.ServerBaseURL+path, &usdRows)
		return resp.StatusCode == http.StatusOK && len(usdRows) > 0
	}, 15*time.Second, 200*time.Millisecond, "coins/markets should return data")

	var eurRows []map[string]interface{}
	resp := getJSON(t, env.ServerBaseURL+path+"&convert_currency=eur", &eurRows)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, eurRows, len(usdRows))

	for i, usdRow := range usdRows {
		eurRow := eurRows[i]
		assert.Equal(t, usdRow["id"], eurRow["id"])

		// spot-scaled fields
		for _, field := range []string{"current_price", "market_cap", "total_volume", "high_24h", "low_24h"} {
			usdValue, ok := usdRow[field].(float64)
			if !ok {
				continue
			}
			assert.InDelta(t, usdValue*RatioFixtureEURNow, eurRow[field].(float64), 1e-6, "field %s", field)
		}

		// honest percent change
		if usdPct, ok := usdRow["price_change_percentage_24h"].(float64); ok {
			expected := ((1+usdPct/100)*RatioFixtureEURNow/RatioFixtureEUR24h - 1) * 100
			assert.InDelta(t, expected, eurRow["price_change_percentage_24h"].(float64), 1e-9)
		}

		// honest absolute delta
		if usdDelta, ok := usdRow["price_change_24h"].(float64); ok {
			usdPrice := usdRow["current_price"].(float64)
			expected := usdPrice*RatioFixtureEURNow - (usdPrice-usdDelta)*RatioFixtureEUR24h
			assert.InDelta(t, expected, eurRow["price_change_24h"].(float64), 1e-6)
		}

		// non-money fields pass through
		assert.Equal(t, usdRow["market_cap_rank"], eurRow["market_cap_rank"])
		assert.Equal(t, usdRow["circulating_supply"], eurRow["circulating_supply"])
		assert.Equal(t, usdRow["last_updated"], eurRow["last_updated"])
	}

	// the Passthrough response is unchanged by the conversion request
	var usdAfter []map[string]interface{}
	resp = getJSON(t, env.ServerBaseURL+path, &usdAfter)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, usdRows, usdAfter)
}

func TestCoinsMarketsConvertCurrencyUnknown(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForDataInitialization(t, env)

	var body map[string]string
	resp := getJSON(t, env.ServerBaseURL+"/api/v1/coins/markets?ids=bitcoin&convert_currency=xyz", &body)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "unsupported convert_currency: xyz", body["error"])
}

// TestSimplePriceConvertCurrency checks that Estimate keys are added alongside
// the Passthrough ones on /api/v1/simple/price
func TestSimplePriceConvertCurrency(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForDataInitialization(t, env)
	waitForConvertedMarkets(t, env)

	const query = "/api/v1/simple/price?ids=bitcoin,ethereum&vs_currencies=usd" +
		"&include_market_cap=true&include_24hr_vol=true&include_24hr_change=true"

	waitForSimplePrices(t, env, query)

	var usdResponse map[string]map[string]float64
	resp := getJSON(t, env.ServerBaseURL+query, &usdResponse)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, usdResponse)

	var converted map[string]map[string]float64
	resp = getJSON(t, env.ServerBaseURL+query+"&convert_currency=eur", &converted)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, converted, len(usdResponse))

	for tokenID, usdRow := range usdResponse {
		row, ok := converted[tokenID]
		require.True(t, ok, "token %s should be present", tokenID)

		// Passthrough keys survive untouched
		assert.Equal(t, usdRow["usd"], row["usd"])
		assert.Equal(t, usdRow["usd_market_cap"], row["usd_market_cap"])
		assert.Equal(t, usdRow["usd_24h_change"], row["usd_24h_change"])

		// Estimate keys are added
		assert.InDelta(t, usdRow["usd"]*RatioFixtureEURNow, row["eur"], 1e-6)
		assert.InDelta(t, usdRow["usd_market_cap"]*RatioFixtureEURNow, row["eur_market_cap"], 1e-3)
		assert.InDelta(t, usdRow["usd_24h_vol"]*RatioFixtureEURNow, row["eur_24h_vol"], 1e-3)

		expected := ((1+usdRow["usd_24h_change"]/100)*RatioFixtureEURNow/RatioFixtureEUR24h - 1) * 100
		assert.InDelta(t, expected, row["eur_24h_change"], 1e-9)
	}
}

// TestSimplePriceConvertCurrencyWithoutBaseRequested checks that the base
// currency is read in for the Estimate but not leaked into the response
func TestSimplePriceConvertCurrencyWithoutBaseRequested(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForDataInitialization(t, env)
	waitForConvertedMarkets(t, env)

	const query = "/api/v1/simple/price?ids=bitcoin&vs_currencies=eur"
	waitForSimplePrices(t, env, query)

	var converted map[string]map[string]float64
	resp := getJSON(t, env.ServerBaseURL+query+"&convert_currency=btc", &converted)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	row, ok := converted["bitcoin"]
	require.True(t, ok)
	assert.Contains(t, row, "eur", "the requested passthrough currency is served")
	assert.Contains(t, row, "btc", "the estimate currency is added")
	assert.NotContains(t, row, "usd", "the base currency was not requested")
}

// TestSimplePriceConvertCurrencyErrors covers the 400 cases
func TestSimplePriceConvertCurrencyErrors(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForDataInitialization(t, env)

	tests := []struct {
		name          string
		query         string
		expectedError string
	}{
		{
			name:          "unknown currency",
			query:         "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=xyz",
			expectedError: "unsupported convert_currency: xyz",
		},
		{
			name:          "ambiguous with vs_currencies",
			query:         "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd,eur&convert_currency=eur",
			expectedError: "convert_currency eur must not be listed in vs_currencies",
		},
		{
			name:          "ambiguous base currency",
			query:         "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&convert_currency=usd",
			expectedError: "convert_currency usd must not be listed in vs_currencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body map[string]string
			resp := getJSON(t, env.ServerBaseURL+tt.query, &body)

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Equal(t, tt.expectedError, body["error"])
		})
	}
}

// TestLeaderboardPricesConvertCurrency checks the realtime Estimate on
// /api/v1/leaderboard/prices
func TestLeaderboardPricesConvertCurrency(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	waitForDataInitialization(t, env)
	waitForConvertedMarkets(t, env)

	// The prices cache is refilled on every price update cycle and can be
	// momentarily empty, so poll until it holds quotes.
	var usdQuotes map[string]map[string]float64
	require.Eventually(t, func() bool {
		usdQuotes = nil
		resp := getJSON(t, env.ServerBaseURL+"/api/v1/leaderboard/prices", &usdQuotes)
		return resp.StatusCode == http.StatusOK && len(usdQuotes) > 0
	}, 15*time.Second, 200*time.Millisecond, "leaderboard prices should become available")

	var eurQuotes map[string]map[string]float64
	resp := getJSON(t, env.ServerBaseURL+"/api/v1/leaderboard/prices?convert_currency=eur", &eurQuotes)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.Len(t, eurQuotes, len(usdQuotes))

	for tokenID, usdQuote := range usdQuotes {
		eurQuote, ok := eurQuotes[tokenID]
		require.True(t, ok, "token %s should be present in the converted response", tokenID)

		assert.InDelta(t, usdQuote["price"]*RatioFixtureEURNow, eurQuote["price"], 1e-6)
		assert.InDelta(t, usdQuote["volume_24h"]*RatioFixtureEURNow, eurQuote["volume_24h"], 1e-3)
		assert.InDelta(t, usdQuote["market_cap"]*RatioFixtureEURNow, eurQuote["market_cap"], 1e-3)

		expectedPct := ((1+usdQuote["percent_change_24h"]/100)*RatioFixtureEURNow/RatioFixtureEUR24h - 1) * 100
		assert.InDelta(t, expectedPct, eurQuote["percent_change_24h"], 1e-9)
	}
}
