package e2etest

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

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
	assert.Contains(t, services, "coingecko", "Services should include 'coingecko'")
	assert.Contains(t, services, "tokens", "Services should include 'tokens'")
}

