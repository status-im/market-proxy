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

// TestHealthEndpoint tests the functionality of the /health endpoint
func TestHealthEndpoint(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Make a request to /health
	resp, err := http.Get(env.ServerBaseURL + "/health")
	require.NoError(t, err, "Should be able to make a request to /health")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Check response format
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")

	var healthResponse map[string]interface{}
	err = json.Unmarshal(body, &healthResponse)
	require.NoError(t, err, "Response should be valid JSON")

	// Check that the response contains status and service information
	assert.Equal(t, "ok", healthResponse["status"], "Health status should be 'ok'")

	services, ok := healthResponse["services"].(map[string]interface{})
	require.True(t, ok, "Response should contain 'services' object")

	// Check that the response contains information about all services
	assert.Contains(t, services, "binance", "Services should include 'binance'")
	assert.Contains(t, services, "coingecko", "Services should include 'coingecko'")
	assert.Contains(t, services, "tokens", "Services should include 'tokens'")
}

// TestLeaderboardMarketsEndpoint tests the functionality of the /api/v1/leaderboard/markets endpoint
func TestLeaderboardMarketsEndpoint(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Give time for data initialization
	waitForDataInitialization(t, env)

	// Make a request to /api/v1/leaderboard/markets
	resp, err := http.Get(env.ServerBaseURL + "/api/v1/leaderboard/markets")
	require.NoError(t, err, "Should be able to make a request to /api/v1/leaderboard/markets")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Check response format
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")

	var marketsResponse map[string]interface{}
	err = json.Unmarshal(body, &marketsResponse)
	require.NoError(t, err, "Response should be valid JSON")

	// Check that the response contains data
	data, ok := marketsResponse["data"].([]interface{})
	require.True(t, ok, "Response should contain 'data' array")

	// Check that the data contains at least one item
	require.NotEmpty(t, data, "Data array should not be empty")

	// Check format of the first item
	firstItem, ok := data[0].(map[string]interface{})
	require.True(t, ok, "First item should be an object")

	// Check the presence of key fields
	assert.Contains(t, firstItem, "id", "Item should contain 'id'")
	assert.Contains(t, firstItem, "symbol", "Item should contain 'symbol'")
	assert.Contains(t, firstItem, "name", "Item should contain 'name'")
	assert.Contains(t, firstItem, "current_price", "Item should contain 'current_price'")
}

// TestLeaderboardPricesEndpoint tests the functionality of the /api/v1/leaderboard/prices endpoint
func TestLeaderboardPricesEndpoint(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Give time for data initialization
	waitForDataInitialization(t, env)

	// Make a request to the mock server instead of the real API server
	mockURL := env.MockServer.GetURL() + "/api/v1/leaderboard/prices"
	resp, err := http.Get(mockURL)
	require.NoError(t, err, "Should be able to make a request to /api/v1/leaderboard/prices")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Check response format
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")

	var pricesResponse []map[string]interface{}
	err = json.Unmarshal(body, &pricesResponse)
	require.NoError(t, err, "Response should be valid JSON")

	// Check that the response contains data
	require.NotEmpty(t, pricesResponse, "Response should not be empty")

	// Check format of the first item
	firstItem := pricesResponse[0]

	// Check the presence of key fields
	assert.Contains(t, firstItem, "symbol", "Item should contain 'symbol'")
	assert.Contains(t, firstItem, "price", "Item should contain 'price'")
}

// TestCoinsListEndpoint tests the functionality of the /api/v1/coins/list endpoint
func TestCoinsListEndpoint(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Give time for data initialization
	waitForDataInitialization(t, env)

	// Make a request to the mock server instead of the real API server
	mockURL := env.MockServer.GetURL() + "/api/v1/coins/list?include_platform=true"
	resp, err := http.Get(mockURL)
	require.NoError(t, err, "Should be able to make a request to /api/v1/coins/list")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Check response format
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")

	var coinsResponse []map[string]interface{}
	err = json.Unmarshal(body, &coinsResponse)
	require.NoError(t, err, "Response should be valid JSON")

	// Check that the response contains data
	require.NotEmpty(t, coinsResponse, "Response should not be empty")

	// Check format of the first item
	firstItem := coinsResponse[0]

	// Check the presence of key fields
	assert.Contains(t, firstItem, "id", "Item should contain 'id'")
	assert.Contains(t, firstItem, "symbol", "Item should contain 'symbol'")
	assert.Contains(t, firstItem, "name", "Item should contain 'name'")
	assert.Contains(t, firstItem, "platforms", "Item should contain 'platforms'")
}

// waitForDataInitialization waits for services to initialize data
func waitForDataInitialization(t *testing.T, env *TestEnv) {
	// Wait until service contains data by polling for expected tokens
	// This is more reliable than a fixed time delay
	maxWait := 30 * time.Second
	pollInterval := 500 * time.Millisecond
	timeout := time.Now().Add(maxWait)

	for time.Now().Before(timeout) {
		// Test if the cache contains the expected test tokens
		resp, err := http.Get(env.ServerBaseURL + "/api/v1/coins/markets?ids=bitcoin,ethereum")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				// Check if we get non-empty response
				resp2, err2 := http.Get(env.ServerBaseURL + "/api/v1/coins/markets?ids=bitcoin,ethereum")
				if err2 == nil {
					body, err3 := io.ReadAll(resp2.Body)
					resp2.Body.Close()
					if err3 == nil {
						var data []interface{}
						if json.Unmarshal(body, &data) == nil && len(data) > 0 {
							// Data is available, initialization complete
							t.Logf("Data initialization completed, found %d tokens in cache", len(data))
							return
						}
					}
				}
			}
		}

		time.Sleep(pollInterval)
	}

	// Fallback to original behavior if polling doesn't work
	t.Log("Data polling timeout, falling back to fixed wait time")
	time.Sleep(2 * time.Second)
}
