package coingecko_markets

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTokensData_Success(t *testing.T) {
	// Create valid test data
	token1 := map[string]interface{}{
		"id":            "bitcoin",
		"symbol":        "btc",
		"name":          "Bitcoin",
		"current_price": 50000.0,
	}
	token2 := map[string]interface{}{
		"id":            "ethereum",
		"symbol":        "eth",
		"name":          "Ethereum",
		"current_price": 3000.0,
	}

	token1Bytes, err := json.Marshal(token1)
	require.NoError(t, err)
	token2Bytes, err := json.Marshal(token2)
	require.NoError(t, err)

	tokensData := [][]byte{token1Bytes, token2Bytes}

	// Test parseTokensData
	marketData, cacheData, err := parseTokensData(tokensData)

	// Assertions
	require.NoError(t, err)
	assert.Len(t, marketData, 2)
	assert.Len(t, cacheData, 2)

	// Check market data content
	assert.Equal(t, token1, marketData[0])
	assert.Equal(t, token2, marketData[1])

	// Check cache data keys and values
	expectedKey1 := getCacheKey("bitcoin")
	expectedKey2 := getCacheKey("ethereum")

	assert.Contains(t, cacheData, expectedKey1)
	assert.Contains(t, cacheData, expectedKey2)
	assert.Equal(t, token1Bytes, cacheData[expectedKey1])
	assert.Equal(t, token2Bytes, cacheData[expectedKey2])
}

func TestParseTokensData_EmptyInput(t *testing.T) {
	tokensData := [][]byte{}

	marketData, cacheData, err := parseTokensData(tokensData)

	require.NoError(t, err)
	assert.Empty(t, marketData)
	assert.Empty(t, cacheData)
}

func TestParseTokensData_InvalidJSON(t *testing.T) {
	// Create invalid JSON data
	invalidJSON := []byte(`{"id": "bitcoin", "symbol": "btc"`)
	tokensData := [][]byte{invalidJSON}

	marketData, cacheData, err := parseTokensData(tokensData)

	// Should skip invalid JSON gracefully
	require.NoError(t, err)
	assert.Empty(t, marketData)
	assert.Empty(t, cacheData)
}

func TestParseTokensData_MissingIDField(t *testing.T) {
	// Create token without ID field
	token := map[string]interface{}{
		"symbol":        "btc",
		"name":          "Bitcoin",
		"current_price": 50000.0,
	}

	tokenBytes, err := json.Marshal(token)
	require.NoError(t, err)
	tokensData := [][]byte{tokenBytes}

	marketData, cacheData, err := parseTokensData(tokensData)

	// Should skip token missing ID field gracefully
	require.NoError(t, err)
	assert.Empty(t, marketData)
	assert.Empty(t, cacheData)
}

func TestParseTokensData_InvalidIDType(t *testing.T) {
	// Create token with non-string ID field
	token := map[string]interface{}{
		"id":            123, // Invalid type (should be string)
		"symbol":        "btc",
		"name":          "Bitcoin",
		"current_price": 50000.0,
	}

	tokenBytes, err := json.Marshal(token)
	require.NoError(t, err)
	tokensData := [][]byte{tokenBytes}

	marketData, cacheData, err := parseTokensData(tokensData)

	// Should skip token with invalid ID type gracefully
	require.NoError(t, err)
	assert.Empty(t, marketData)
	assert.Empty(t, cacheData)
}

func TestParseTokensData_EmptyIDField(t *testing.T) {
	// Create token with empty ID field
	token := map[string]interface{}{
		"id":            "", // Empty string ID
		"symbol":        "btc",
		"name":          "Bitcoin",
		"current_price": 50000.0,
	}

	tokenBytes, err := json.Marshal(token)
	require.NoError(t, err)
	tokensData := [][]byte{tokenBytes}

	marketData, cacheData, err := parseTokensData(tokensData)

	// Should skip token with empty ID gracefully
	require.NoError(t, err)
	assert.Empty(t, marketData)
	assert.Empty(t, cacheData)
}

func TestParseTokensData_InvalidMarketDataFormat(t *testing.T) {
	// Create non-map JSON data
	invalidData := []byte(`"this is not a map"`)
	tokensData := [][]byte{invalidData}

	marketData, cacheData, err := parseTokensData(tokensData)

	// Should skip invalid data format gracefully
	require.NoError(t, err)
	assert.Empty(t, marketData)
	assert.Empty(t, cacheData)
}

func TestParseTokensData_FirstTokenValid_SecondTokenInvalid(t *testing.T) {
	// Create one valid and one invalid token
	validToken := map[string]interface{}{
		"id":            "bitcoin",
		"symbol":        "btc",
		"name":          "Bitcoin",
		"current_price": 50000.0,
	}

	validTokenBytes, err := json.Marshal(validToken)
	require.NoError(t, err)

	invalidJSON := []byte(`{"id": "ethereum"`) // Missing closing brace

	tokensData := [][]byte{validTokenBytes, invalidJSON}

	marketData, cacheData, err := parseTokensData(tokensData)

	// Should process valid token and skip invalid one gracefully
	require.NoError(t, err)
	assert.Len(t, marketData, 1)
	assert.Len(t, cacheData, 1)

	// Check that the valid token was processed
	expectedKey := getCacheKey("bitcoin")
	assert.Contains(t, cacheData, expectedKey)
}

func TestParseTokensData_CacheKeyGeneration(t *testing.T) {
	// Test that cache keys are generated correctly
	token := map[string]interface{}{
		"id":            "test-token-123",
		"symbol":        "tst",
		"name":          "Test Token",
		"current_price": 1.0,
	}

	tokenBytes, err := json.Marshal(token)
	require.NoError(t, err)
	tokensData := [][]byte{tokenBytes}

	marketData, cacheData, err := parseTokensData(tokensData)

	require.NoError(t, err)
	assert.Len(t, marketData, 1)
	assert.Len(t, cacheData, 1)

	// Check that the cache key is generated correctly
	expectedKey := getCacheKey("test-token-123")
	assert.Contains(t, cacheData, expectedKey)
	assert.True(t, strings.HasPrefix(expectedKey, CACHE_KEY_PREFIX))
	assert.Contains(t, expectedKey, "test-token-123")
}

func TestParseTokensData_MultipleValidTokens(t *testing.T) {
	// Test with multiple valid tokens to ensure all are processed
	tokens := []map[string]interface{}{
		{"id": "bitcoin", "symbol": "btc", "name": "Bitcoin"},
		{"id": "ethereum", "symbol": "eth", "name": "Ethereum"},
		{"id": "cardano", "symbol": "ada", "name": "Cardano"},
	}

	var tokensData [][]byte
	for _, token := range tokens {
		tokenBytes, err := json.Marshal(token)
		require.NoError(t, err)
		tokensData = append(tokensData, tokenBytes)
	}

	marketData, cacheData, err := parseTokensData(tokensData)

	require.NoError(t, err)
	assert.Len(t, marketData, 3)
	assert.Len(t, cacheData, 3)

	// Verify all tokens are processed
	for i, expectedToken := range tokens {
		assert.Equal(t, expectedToken, marketData[i])

		expectedKey := getCacheKey(expectedToken["id"].(string))
		assert.Contains(t, cacheData, expectedKey)
	}
}

func TestParsePagesData_PageMapping(t *testing.T) {
	// Create test data with multiple pages
	page1Data := [][]byte{
		[]byte(`{"id": "bitcoin", "symbol": "btc", "name": "Bitcoin"}`),
		[]byte(`{"id": "ethereum", "symbol": "eth", "name": "Ethereum"}`),
	}
	page2Data := [][]byte{
		[]byte(`{"id": "cardano", "symbol": "ada", "name": "Cardano"}`),
	}
	page3Data := [][]byte{
		[]byte(`{"id": "polygon", "symbol": "matic", "name": "Polygon"}`),
		[]byte(`{"id": "solana", "symbol": "sol", "name": "Solana"}`),
	}

	// Create pages data: page 3, page 1, page 2
	pagesData := []PageData{
		{Page: 3, Data: page3Data},
		{Page: 1, Data: page1Data},
		{Page: 2, Data: page2Data},
	}

	// Test parsePagesData
	pageMapping, cacheData, err := parsePagesData(pagesData)

	// Assertions
	require.NoError(t, err)
	assert.Len(t, pageMapping, 3)
	assert.Len(t, cacheData, 3)

	// Check that all pages are present in pageMapping
	assert.Contains(t, pageMapping, 1)
	assert.Contains(t, pageMapping, 2)
	assert.Contains(t, pageMapping, 3)

	// Verify the data content for each page
	assert.Equal(t, page1Data, pageMapping[1])
	assert.Equal(t, page2Data, pageMapping[2])
	assert.Equal(t, page3Data, pageMapping[3])

	// Check cache data keys
	expectedKey1 := createPageCacheKey(1)
	expectedKey2 := createPageCacheKey(2)
	expectedKey3 := createPageCacheKey(3)

	assert.Contains(t, cacheData, expectedKey1)
	assert.Contains(t, cacheData, expectedKey2)
	assert.Contains(t, cacheData, expectedKey3)

	// Verify cached data matches original data
	var cachedPage1Data [][]byte
	err = json.Unmarshal(cacheData[expectedKey1], &cachedPage1Data)
	require.NoError(t, err)
	assert.Equal(t, page1Data, cachedPage1Data)

	var cachedPage2Data [][]byte
	err = json.Unmarshal(cacheData[expectedKey2], &cachedPage2Data)
	require.NoError(t, err)
	assert.Equal(t, page2Data, cachedPage2Data)

	var cachedPage3Data [][]byte
	err = json.Unmarshal(cacheData[expectedKey3], &cachedPage3Data)
	require.NoError(t, err)
	assert.Equal(t, page3Data, cachedPage3Data)
}

func TestParsePagesData_EmptyInput(t *testing.T) {
	pagesData := []PageData{}

	pageMapping, cacheData, err := parsePagesData(pagesData)

	require.NoError(t, err)
	assert.Empty(t, pageMapping)
	assert.Empty(t, cacheData)
}

func TestParsePagesData_SinglePage(t *testing.T) {
	pageData := [][]byte{
		[]byte(`{"id": "bitcoin", "symbol": "btc", "name": "Bitcoin"}`),
	}

	pagesData := []PageData{
		{Page: 5, Data: pageData},
	}

	pageMapping, cacheData, err := parsePagesData(pagesData)

	require.NoError(t, err)
	assert.Len(t, pageMapping, 1)
	assert.Len(t, cacheData, 1)

	// Check page mapping
	assert.Contains(t, pageMapping, 5)
	assert.Equal(t, pageData, pageMapping[5])

	// Check cache data
	expectedKey := createPageCacheKey(5)
	assert.Contains(t, cacheData, expectedKey)
}
