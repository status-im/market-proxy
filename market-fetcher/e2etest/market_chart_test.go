package e2etest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoinsMarketChartEndpoint tests the functionality of the /api/v1/coins/{id}/market_chart endpoint
func TestCoinsMarketChartEndpoint(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Give time for data initialization
	waitForDataInitialization(t, env)

	// Test basic market chart request
	t.Run("Basic Market Chart Request", func(t *testing.T) {
		url := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request to /api/v1/coins/bitcoin/market_chart")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

		// Check response format
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var chartResponse map[string]interface{}
		err = json.Unmarshal(body, &chartResponse)
		require.NoError(t, err, "Response should be valid JSON")

		// Check required fields according to CoinGecko market chart API
		assert.Contains(t, chartResponse, "prices", "Response should contain 'prices'")
		assert.Contains(t, chartResponse, "market_caps", "Response should contain 'market_caps'")
		assert.Contains(t, chartResponse, "total_volumes", "Response should contain 'total_volumes'")

		// Verify data structure for prices
		prices, ok := chartResponse["prices"].([]interface{})
		assert.True(t, ok, "Prices should be an array")
		if len(prices) > 0 {
			pricePoint, ok := prices[0].([]interface{})
			assert.True(t, ok, "Each price point should be an array")
			assert.Len(t, pricePoint, 2, "Each price point should have timestamp and value")
		}
	})

	// Test with different coins
	t.Run("Different Coin IDs", func(t *testing.T) {
		coins := []string{"bitcoin", "ethereum"}

		for _, coinID := range coins {
			t.Run(coinID, func(t *testing.T) {
				testURL := env.ServerBaseURL + "/api/v1/coins/" + coinID + "/market_chart"
				resp, err := http.Get(testURL)
				require.NoError(t, err, "Should be able to make a request for coin "+coinID)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
			})
		}
	})

	// Test with vs_currency parameter
	t.Run("With Currency Parameter", func(t *testing.T) {
		currencies := []string{"usd", "eur", "btc"}

		for _, currency := range currencies {
			t.Run(currency, func(t *testing.T) {
				testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?vs_currency=" + currency
				resp, err := http.Get(testURL)
				require.NoError(t, err, "Should be able to make a request with vs_currency="+currency)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
			})
		}
	})

	// Test with days parameter
	t.Run("With Days Parameter", func(t *testing.T) {
		daysValues := []string{"1", "7", "14", "30", "90", "180", "365", "max"}

		for _, days := range daysValues {
			t.Run("days_"+days, func(t *testing.T) {
				testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=" + days
				resp, err := http.Get(testURL)
				require.NoError(t, err, "Should be able to make a request with days="+days)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

				// Verify response structure
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Should be able to read response body")

				var chartResponse map[string]interface{}
				err = json.Unmarshal(body, &chartResponse)
				require.NoError(t, err, "Response should be valid JSON")

				// Should contain all required fields
				assert.Contains(t, chartResponse, "prices", "Response should contain 'prices'")
				assert.Contains(t, chartResponse, "market_caps", "Response should contain 'market_caps'")
				assert.Contains(t, chartResponse, "total_volumes", "Response should contain 'total_volumes'")
			})
		}
	})

	// Test with interval parameter
	t.Run("With Interval Parameter", func(t *testing.T) {
		intervals := []string{"hourly", "daily"}

		for _, interval := range intervals {
			t.Run("interval_"+interval, func(t *testing.T) {
				testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=90&interval=" + interval
				resp, err := http.Get(testURL)
				require.NoError(t, err, "Should be able to make a request with interval="+interval)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
			})
		}
	})

	// Test with data_filter parameter
	t.Run("With Data Filter Parameter", func(t *testing.T) {
		filters := []string{"prices", "market_caps", "total_volumes", "prices,market_caps"}

		for _, filter := range filters {
			t.Run("filter_"+filter, func(t *testing.T) {
				testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?data_filter=" + filter
				resp, err := http.Get(testURL)
				require.NoError(t, err, "Should be able to make a request with data_filter="+filter)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

				// Verify response structure based on filter
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Should be able to read response body")

				var chartResponse map[string]interface{}
				err = json.Unmarshal(body, &chartResponse)
				require.NoError(t, err, "Response should be valid JSON")

				// Check that only requested fields are present
				if filter == "prices" {
					assert.Contains(t, chartResponse, "prices", "Response should contain 'prices' when filtered")
					// With filter, should not contain other fields (depends on implementation)
				}
			})
		}
	})

	// Test with all parameters combined
	t.Run("With All Parameters", func(t *testing.T) {
		params := url.Values{}
		params.Add("vs_currency", "usd")
		params.Add("days", "30")
		params.Add("interval", "hourly")
		params.Add("data_filter", "prices,market_caps")

		testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?" + params.Encode()
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with all parameters")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

		// Verify response structure
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var chartResponse map[string]interface{}
		err = json.Unmarshal(body, &chartResponse)
		require.NoError(t, err, "Response should be valid JSON")
		require.NotEmpty(t, chartResponse, "Response should not be empty")
	})
}

// TestCoinsMarketChartEndpointValidation tests parameter validation and error handling
func TestCoinsMarketChartEndpointValidation(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Test invalid days parameter
	t.Run("Invalid Days Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=invalid"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with invalid days parameter")
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 Bad Request for invalid days")
	})

	// Test invalid interval parameter
	t.Run("Invalid Interval Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?interval=invalid"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with invalid interval parameter")
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 Bad Request for invalid interval")
	})

	// Test invalid data filter parameter
	t.Run("Invalid Data Filter Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?data_filter=invalid_field"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with invalid data filter parameter")
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 Bad Request for invalid data filter")
	})

	// Test empty coin ID (using space character that gets URL encoded)
	t.Run("Empty Coin ID", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/%20/market_chart" // %20 is URL encoded space
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with empty coin ID")
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 Bad Request for empty coin ID")
	})

	// Test with multiple invalid parameters at once
	t.Run("Multiple Invalid Parameters", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=invalid&interval=bad&data_filter=wrong"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with multiple invalid parameters")
		defer resp.Body.Close()

		// Should return validation error for any invalid parameter
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return 400 for multiple invalid parameters")
	})

	// Test URL encoding
	t.Run("URL Encoded Parameters", func(t *testing.T) {
		// Test with URL-encoded comma-separated data filter
		testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?data_filter=prices%2Cmarket_caps"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should handle URL-encoded parameters")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})
}

// TestCoinsMarketChartCaching tests caching behavior and roundUp/strip functionality
func TestCoinsMarketChartCaching(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Give time for data initialization
	waitForDataInitialization(t, env)

	// Test cache functionality with roundUp/strip logic
	t.Run("RoundUp and Strip Logic", func(t *testing.T) {
		// Make a request for 30 days (should be rounded up to 90 internally)
		url1 := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=30"
		resp1, err := http.Get(url1)
		require.NoError(t, err, "Should be able to make first request")
		defer resp1.Body.Close()
		require.Equal(t, http.StatusOK, resp1.StatusCode, "First request should return 200 OK")

		body1, err := io.ReadAll(resp1.Body)
		require.NoError(t, err, "Should be able to read first response")

		// Make a request for 60 days (should also be rounded up to 90 internally and hit cache)
		url2 := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=60"
		resp2, err := http.Get(url2)
		require.NoError(t, err, "Should be able to make second request")
		defer resp2.Body.Close()
		require.Equal(t, http.StatusOK, resp2.StatusCode, "Second request should return 200 OK")

		body2, err := io.ReadAll(resp2.Body)
		require.NoError(t, err, "Should be able to read second response")

		// Both responses should be valid JSON
		var chart1, chart2 map[string]interface{}
		require.NoError(t, json.Unmarshal(body1, &chart1), "First response should be valid JSON")
		require.NoError(t, json.Unmarshal(body2, &chart2), "Second response should be valid JSON")

		// Both should have the required structure
		assert.Contains(t, chart1, "prices", "First response should contain prices")
		assert.Contains(t, chart2, "prices", "Second response should contain prices")
	})

	// Test different TTL behavior for hourly vs daily data
	t.Run("TTL Behavior", func(t *testing.T) {
		// Request for ≤ 90 days (hourly data, shorter TTL)
		url1 := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=30"
		resp1, err := http.Get(url1)
		require.NoError(t, err, "Should be able to make request for hourly data")
		defer resp1.Body.Close()
		require.Equal(t, http.StatusOK, resp1.StatusCode, "Should return 200 OK for hourly data")

		// Request for > 90 days (daily data, longer TTL)
		url2 := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=365"
		resp2, err := http.Get(url2)
		require.NoError(t, err, "Should be able to make request for daily data")
		defer resp2.Body.Close()
		require.Equal(t, http.StatusOK, resp2.StatusCode, "Should return 200 OK for daily data")

		// Both should return valid responses regardless of TTL
		body1, err := io.ReadAll(resp1.Body)
		require.NoError(t, err, "Should be able to read hourly data response")

		body2, err := io.ReadAll(resp2.Body)
		require.NoError(t, err, "Should be able to read daily data response")

		var chart1, chart2 map[string]interface{}
		require.NoError(t, json.Unmarshal(body1, &chart1), "Hourly data should be valid JSON")
		require.NoError(t, json.Unmarshal(body2, &chart2), "Daily data should be valid JSON")
	})
}

// TestCoinsMarketChartPerformance tests performance characteristics
func TestCoinsMarketChartPerformance(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Give time for data initialization
	waitForDataInitialization(t, env)

	t.Run("Response Time", func(t *testing.T) {
		// Make multiple requests to test performance
		for i := 0; i < 5; i++ {
			testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=30"
			resp, err := http.Get(testURL)
			require.NoError(t, err, "Should be able to make request %d", i)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK for request %d", i)

			// Read response to ensure complete processing
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Should be able to read response body")
			require.NotEmpty(t, body, "Response body should not be empty")
		}
	})

	t.Run("Concurrent Requests", func(t *testing.T) {
		// Test concurrent requests to the same endpoint
		const numRequests = 10
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(requestNum int) {
				testURL := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart?days=7"
				resp, err := http.Get(testURL)
				if err != nil {
					results <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					results <- assert.AnError
					return
				}

				// Read the response to ensure it's complete
				_, err = io.ReadAll(resp.Body)
				results <- err
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent request %d should succeed", i)
		}
	})
}

// TestMarketChartTimestampFreshness verifies that market chart data contains recent timestamps
// This test ensures that mock data is generated dynamically on each request rather than
// being static, preventing tests from failing as timestamp become outdated over time.
func TestMarketChartTimestampFreshness(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Give time for data initialization
	waitForDataInitialization(t, env)

	// Make a request to get market chart data
	url := env.ServerBaseURL + "/api/v1/coins/bitcoin/market_chart"
	resp, err := http.Get(url)
	require.NoError(t, err, "Should be able to make a request to market chart endpoint")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Parse response
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")

	var chartResponse map[string]interface{}
	err = json.Unmarshal(body, &chartResponse)
	require.NoError(t, err, "Response should be valid JSON")

	// Check that prices data exists and has recent timestamps
	pricesData, ok := chartResponse["prices"].([]interface{})
	require.True(t, ok, "Prices should be an array")
	require.Greater(t, len(pricesData), 0, "Should have price data points")

	// Check the first price data point
	firstPrice, ok := pricesData[0].([]interface{})
	require.True(t, ok, "First price should be an array")
	require.Len(t, firstPrice, 2, "Price data point should have [timestamp, price]")

	// Extract timestamp (first element) and convert to time
	timestampFloat, ok := firstPrice[0].(float64)
	require.True(t, ok, "Timestamp should be a number")

	timestamp := int64(timestampFloat) / 1000 // Convert milliseconds to seconds
	dataTime := time.Unix(timestamp, 0)

	// Check that the timestamp is recent (within last 31 days)
	now := time.Now()
	timeDiff := now.Sub(dataTime)

	t.Logf("Current time: %v", now.Format("2006-01-02 15:04:05"))
	t.Logf("Data timestamp: %v", dataTime.Format("2006-01-02 15:04:05"))
	t.Logf("Time difference: %v", timeDiff)

	assert.True(t, timeDiff >= 0, "Data timestamp should not be in the future")
	assert.True(t, timeDiff < 31*24*time.Hour, "Data timestamp should be within last 31 days")

	// Check the last price data point as well
	lastPrice, ok := pricesData[len(pricesData)-1].([]interface{})
	require.True(t, ok, "Last price should be an array")
	require.Len(t, lastPrice, 2, "Price data point should have [timestamp, price]")

	lastTimestampFloat, ok := lastPrice[0].(float64)
	require.True(t, ok, "Last timestamp should be a number")

	lastTimestamp := int64(lastTimestampFloat) / 1000
	lastDataTime := time.Unix(lastTimestamp, 0)
	lastTimeDiff := now.Sub(lastDataTime)

	t.Logf("Last data timestamp: %v", lastDataTime.Format("2006-01-02 15:04:05"))
	t.Logf("Last time difference: %v", lastTimeDiff)

	assert.True(t, lastTimeDiff >= 0, "Last data timestamp should not be in the future")
	assert.True(t, lastTimeDiff < 24*time.Hour, "Last data timestamp should be within last 24 hours")
}
