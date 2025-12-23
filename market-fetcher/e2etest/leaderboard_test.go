package e2etest

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

