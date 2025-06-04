package e2etest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSimplePriceEndpoint tests the functionality of the /api/v1/simple/price endpoint
func TestSimplePriceEndpoint(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Test basic price request
	t.Run("Basic Price Request", func(t *testing.T) {
		url := env.ServerBaseURL + "/api/v1/simple/price?ids=bitcoin,ethereum&vs_currencies=usd,eur"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request to /api/v1/simple/price")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

		// Check response format
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var priceResponse map[string]interface{}
		err = json.Unmarshal(body, &priceResponse)
		require.NoError(t, err, "Response should be valid JSON")

		// Since the cache is empty and loader returns empty data, we expect empty response
		// But the structure should be correct
		assert.IsType(t, map[string]interface{}{}, priceResponse, "Response should be a map")
	})

	// Test with market cap inclusion
	t.Run("Price Request with Market Cap", func(t *testing.T) {
		url := env.ServerBaseURL + "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&include_market_cap=true"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request with market cap")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var priceResponse map[string]interface{}
		err = json.Unmarshal(body, &priceResponse)
		require.NoError(t, err, "Response should be valid JSON")
	})

	// Test with all additional parameters
	t.Run("Price Request with All Parameters", func(t *testing.T) {
		url := env.ServerBaseURL + "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&include_market_cap=true&include_24hr_vol=true&include_24hr_change=true&include_last_updated_at=true&precision=2"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request with all parameters")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var priceResponse map[string]interface{}
		err = json.Unmarshal(body, &priceResponse)
		require.NoError(t, err, "Response should be valid JSON")
	})

	// Test missing required parameters
	t.Run("Missing IDs Parameter", func(t *testing.T) {
		url := env.ServerBaseURL + "/api/v1/simple/price?vs_currencies=usd"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 for missing ids")
	})

	t.Run("Missing Currencies Parameter", func(t *testing.T) {
		url := env.ServerBaseURL + "/api/v1/simple/price?ids=bitcoin"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request")
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 for missing vs_currencies")
	})

	// Test boolean parameter parsing
	t.Run("Boolean Parameters", func(t *testing.T) {
		// Test with false values
		url := env.ServerBaseURL + "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&include_market_cap=false&include_24hr_vol=false"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request with false booleans")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

		// Test with true values
		url = env.ServerBaseURL + "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd&include_market_cap=true&include_24hr_vol=true"
		resp, err = http.Get(url)
		require.NoError(t, err, "Should be able to make a request with true booleans")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})

	// Test multiple tokens and currencies
	t.Run("Multiple Tokens and Currencies", func(t *testing.T) {
		url := env.ServerBaseURL + "/api/v1/simple/price?ids=bitcoin,ethereum,cardano&vs_currencies=usd,eur,gbp"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request with multiple tokens and currencies")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var priceResponse map[string]interface{}
		err = json.Unmarshal(body, &priceResponse)
		require.NoError(t, err, "Response should be valid JSON")
	})

	// Test edge cases
	t.Run("Empty Response Handling", func(t *testing.T) {
		// Test with non-existent token ID
		url := env.ServerBaseURL + "/api/v1/simple/price?ids=nonexistent-token&vs_currencies=usd"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should be able to make a request with non-existent token")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK even for non-existent tokens")

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var priceResponse map[string]interface{}
		err = json.Unmarshal(body, &priceResponse)
		require.NoError(t, err, "Response should be valid JSON")

		// Should be empty since loader returns empty data
		assert.Len(t, priceResponse, 0, "Response should be empty for non-existent tokens")
	})

	// Test URL encoding
	t.Run("URL Encoding", func(t *testing.T) {
		// Test with special characters in parameters
		url := env.ServerBaseURL + "/api/v1/simple/price?ids=bitcoin%2Cethereum&vs_currencies=usd"
		resp, err := http.Get(url)
		require.NoError(t, err, "Should handle URL-encoded parameters")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")
	})
}

// TestSimplePriceEndpointPerformance tests performance aspects of the price endpoint
func TestSimplePriceEndpointPerformance(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	t.Run("Large Request Handling", func(t *testing.T) {
		// Test with many tokens and currencies
		tokens := "bitcoin,ethereum,cardano,polkadot,chainlink,litecoin,bitcoin-cash,stellar,eos,tron"
		currencies := "usd,eur,gbp,jpy,cad,aud,chf,cny,inr,krw"
		url := env.ServerBaseURL + "/api/v1/simple/price?ids=" + tokens + "&vs_currencies=" + currencies

		resp, err := http.Get(url)
		require.NoError(t, err, "Should handle large requests")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK for large requests")

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err, "Should be able to read response body")

		var priceResponse map[string]interface{}
		err = json.Unmarshal(body, &priceResponse)
		require.NoError(t, err, "Response should be valid JSON")
	})

	t.Run("Concurrent Requests", func(t *testing.T) {
		// Test multiple concurrent requests
		numRequests := 5
		responseChan := make(chan *http.Response, numRequests)
		errorChan := make(chan error, numRequests)

		url := env.ServerBaseURL + "/api/v1/simple/price?ids=bitcoin&vs_currencies=usd"

		// Send concurrent requests
		for i := 0; i < numRequests; i++ {
			go func() {
				resp, err := http.Get(url)
				if err != nil {
					errorChan <- err
					return
				}
				responseChan <- resp
			}()
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
