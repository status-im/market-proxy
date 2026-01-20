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

	platforms := []string{
		"ethereum",
		"optimistic-ethereum",
		"arbitrum-one",
		"base",
		"linea",
		"polygon-zkevm",
		"unichain",
		"katana",
		"ink",
		"abstract",
		"zksync",
		"soneium",
		"scroll",
		"blast",
		"binance-smart-chain",
	}

	for _, platform := range platforms {
		// Wait for token list data to be available
		waitForTokenListData(t, env, platform)

		resp, err := http.Get(env.ServerBaseURL + "/api/v1/token_lists/" + platform + "/all.json")
		require.NoError(t, err, "Should be able to make a request for platform: %s", platform)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err, "Should be able to read response body for platform: %s", platform)

		require.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK for platform: %s", platform)

		// Check response format
		var tokenListResponse map[string]interface{}
		err = json.Unmarshal(body, &tokenListResponse)
		require.NoError(t, err, "Response should be valid JSON for platform: %s", platform)

		// Check that the response contains required fields
		assert.Contains(t, tokenListResponse, "name", "Response should contain 'name' for platform: %s", platform)
		assert.Contains(t, tokenListResponse, "version", "Response should contain 'version' for platform: %s", platform)
		assert.Contains(t, tokenListResponse, "tokens", "Response should contain 'tokens' for platform: %s", platform)

		// Check that tokens is an array
		tokens, ok := tokenListResponse["tokens"].([]interface{})
		require.True(t, ok, "Tokens should be an array for platform: %s", platform)

		// If there are tokens, check the format of the first one
		if len(tokens) > 0 {
			firstToken, ok := tokens[0].(map[string]interface{})
			require.True(t, ok, "First token should be an object for platform: %s", platform)
			assert.Contains(t, firstToken, "chainId", "Token should contain 'chainId' for platform: %s", platform)
			assert.Contains(t, firstToken, "address", "Token should contain 'address' for platform: %s", platform)
			assert.Contains(t, firstToken, "symbol", "Token should contain 'symbol' for platform: %s", platform)
			assert.Contains(t, firstToken, "name", "Token should contain 'name' for platform: %s", platform)
			assert.Contains(t, firstToken, "decimals", "Token should contain 'decimals' for platform: %s", platform)
		}
	}
}
