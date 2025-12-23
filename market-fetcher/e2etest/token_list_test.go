package e2etest

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTokenListEndpoint tests the functionality of the /api/v1/token_lists/{platform}/all.json endpoint
func TestTokenListEndpoint(t *testing.T) {
	env := SetupTest(t)
	defer env.TearDown()

	// Wait for token list data to be available
	waitForTokenListData(t, env, "linea")

	// Test Linea token list endpoint
	resp, err := http.Get(env.ServerBaseURL + "/api/v1/token_lists/linea/all.json")
	require.NoError(t, err, "Should be able to make a request to /api/v1/token_lists/linea/all.json")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Check response format
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should be able to read response body")

	var tokenListResponse map[string]interface{}
	err = json.Unmarshal(body, &tokenListResponse)
	require.NoError(t, err, "Response should be valid JSON")

	// Check that the response contains required fields
	assert.Contains(t, tokenListResponse, "name", "Response should contain 'name'")
	assert.Contains(t, tokenListResponse, "version", "Response should contain 'version'")
	assert.Contains(t, tokenListResponse, "tokens", "Response should contain 'tokens'")

	// Check that tokens is an array
	tokens, ok := tokenListResponse["tokens"].([]interface{})
	require.True(t, ok, "Tokens should be an array")

	// If there are tokens, check the format of the first one
	if len(tokens) > 0 {
		firstToken, ok := tokens[0].(map[string]interface{})
		require.True(t, ok, "First token should be an object")
		assert.Contains(t, firstToken, "chainId", "Token should contain 'chainId'")
		assert.Contains(t, firstToken, "address", "Token should contain 'address'")
		assert.Contains(t, firstToken, "symbol", "Token should contain 'symbol'")
		assert.Contains(t, firstToken, "name", "Token should contain 'name'")
		assert.Contains(t, firstToken, "decimals", "Token should contain 'decimals'")
	}

	// Test other platforms as well
	platforms := []string{"ethereum", "base", "arbitrum-one"}
	for _, platform := range platforms {
		resp, err := http.Get(env.ServerBaseURL + "/api/v1/token_lists/" + platform + "/all.json")
		require.NoError(t, err, "Should be able to make a request for platform: %s", platform)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK for platform: %s", platform)
	}
}

