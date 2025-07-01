package e2etest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoinsMarketsEndpoint tests the functionality of the /api/v1/coins/markets endpoint
func TestCoinsMarketsEndpoint(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Give time for data initialization
	waitForDataInitialization(t, env)

	// Test basic markets request
	t.Run("Basic Markets Request", func(t *testing.T) {
		// Use specific IDs that should be available in the mock data
		url := env.ServerBaseURL + "/api/v1/coins/markets?ids=bitcoin,ethereum"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request to /api/v1/coins/markets")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

		// Check response format
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var marketsResponse []map[string]interface{}
		err = json.Unmarshal(body, &marketsResponse)
		require.NoError(t, err, "Response should be valid JSON")

		// Should have at least one item
		require.NotEmpty(t, marketsResponse, "Response should not be empty")

		// Check format of the first item
		firstItem := marketsResponse[0]

		// Check required fields according to CoinGecko markets API
		assert.Contains(t, firstItem, "id", "Item should contain 'id'")
		assert.Contains(t, firstItem, "symbol", "Item should contain 'symbol'")
		assert.Contains(t, firstItem, "name", "Item should contain 'name'")
		assert.Contains(t, firstItem, "current_price", "Item should contain 'current_price'")
		assert.Contains(t, firstItem, "market_cap", "Item should contain 'market_cap'")
		assert.Contains(t, firstItem, "market_cap_rank", "Item should contain 'market_cap_rank'")
		assert.Contains(t, firstItem, "total_volume", "Item should contain 'total_volume'")
		assert.Contains(t, firstItem, "price_change_percentage_24h", "Item should contain 'price_change_percentage_24h'")
	})

	// Test with vs_currency parameter
	t.Run("With Currency Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?vs_currency=usd"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with vs_currency parameter")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})

	// Test with order parameter
	t.Run("With Order Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?order=market_cap_desc"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with order parameter")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})

	// Test with pagination parameters
	t.Run("With Pagination Parameters", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?page=1&per_page=10"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with pagination parameters")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})

	// Test with specific IDs
	t.Run("With Specific IDs", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?ids=bitcoin,ethereum"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with specific IDs")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})

	// Test with category parameter
	t.Run("With Category Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?category=layer-1"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with category parameter")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})

	// Test with sparkline parameter
	t.Run("With Sparkline Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?sparkline=true"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with sparkline parameter")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})

	// Test with price change percentage parameter
	t.Run("With Price Change Percentage Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?price_change_percentage=1h,24h,7d"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with price_change_percentage parameter")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})

	// Test with all parameters combined
	t.Run("With All Parameters", func(t *testing.T) {
		params := url.Values{}
		params.Add("ids", "bitcoin,ethereum") // Add specific IDs
		params.Add("vs_currency", "usd")
		params.Add("order", "market_cap_desc")
		params.Add("page", "1")
		params.Add("per_page", "50")
		params.Add("sparkline", "false")
		params.Add("price_change_percentage", "1h,24h,7d")

		testURL := env.ServerBaseURL + "/api/v1/coins/markets?" + params.Encode()
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with all parameters")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

		// Verify response structure
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var marketsResponse []map[string]interface{}
		err = json.Unmarshal(body, &marketsResponse)
		require.NoError(t, err, "Response should be valid JSON")
		require.NotEmpty(t, marketsResponse, "Response should not be empty")
	})
}

// TestCoinsMarketsEndpointValidation tests parameter validation for the markets endpoint
func TestCoinsMarketsEndpointValidation(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Test with invalid page parameter (should handle gracefully)
	t.Run("Invalid Page Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?page=0"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with invalid page parameter")
		defer resp.Body.Close()

		// Should still return 200, as invalid parameters are typically ignored
		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK even with invalid page")
	})

	// Test with invalid per_page parameter (should handle gracefully)
	t.Run("Invalid Per Page Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?per_page=-1"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with invalid per_page parameter")
		defer resp.Body.Close()

		// Should still return 200, as invalid parameters are typically ignored
		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK even with invalid per_page")
	})

	// Test with invalid boolean parameter
	t.Run("Invalid Boolean Parameter", func(t *testing.T) {
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?sparkline=invalid"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should be able to make a request with invalid boolean parameter")
		defer resp.Body.Close()

		// Should still return 200, as invalid boolean parameters are typically ignored
		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK even with invalid boolean")
	})

	// Test with URL encoding
	t.Run("URL Encoded Parameters", func(t *testing.T) {
		// Test with URL-encoded comma-separated IDs
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?ids=bitcoin%2Cethereum"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should handle URL-encoded parameters")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})
}

// TestCoinsMarketsEndpointPerformance tests performance aspects of the markets endpoint
func TestCoinsMarketsEndpointPerformance(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	t.Run("Large Request Handling", func(t *testing.T) {
		// Test with large per_page value
		testURL := env.ServerBaseURL + "/api/v1/coins/markets?per_page=250"
		resp, err := http.Get(testURL)
		require.NoError(t, err, "Should handle large per_page requests")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK for large requests")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var marketsResponse []map[string]interface{}
		err = json.Unmarshal(body, &marketsResponse)
		require.NoError(t, err, "Response should be valid JSON")
	})

	t.Run("Concurrent Requests", func(t *testing.T) {
		// Test multiple concurrent requests
		numRequests := 5
		responseChan := make(chan *http.Response, numRequests)
		errorChan := make(chan error, numRequests)

		testURL := env.ServerBaseURL + "/api/v1/coins/markets?vs_currency=usd&order=market_cap_desc"

		// Send concurrent requests
		for i := 0; i < numRequests; i++ {
			go func(requestID int) {
				// Add unique parameter to avoid caching issues
				url := testURL + "&page=" + strconv.Itoa(requestID+1)
				resp, err := http.Get(url)
				if err != nil {
					errorChan <- err
					return
				}
				responseChan <- resp
			}(i)
		}

		// Collect responses
		for i := 0; i < numRequests; i++ {
			select {
			case resp := <-responseChan:
				assert.Equal(t, http.StatusOK, resp.StatusCode, "All concurrent requests should succeed")
				resp.Body.Close()
			case err := <-errorChan:
				t.Errorf("Concurrent request failed: %v", err)
			}
		}
	})
}
