package e2etest

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

