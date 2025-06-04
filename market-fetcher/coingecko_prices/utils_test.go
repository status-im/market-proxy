package coingecko_prices

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCacheKeys(t *testing.T) {
	// Test single token
	params1 := PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd"},
	}
	keys1 := createCacheKeys(params1)
	assert.Len(t, keys1, 1)
	assert.Equal(t, "simple_price:bitcoin", keys1[0])

	// Test multiple currencies (should not affect cache key)
	params2 := PriceParams{
		IDs:        []string{"bitcoin"},
		Currencies: []string{"usd", "eur"},
	}
	keys2 := createCacheKeys(params2)
	assert.Len(t, keys2, 1)
	assert.Equal(t, "simple_price:bitcoin", keys2[0])
	assert.Equal(t, keys1[0], keys2[0]) // Should be same since currencies are not in key

	// Test different token
	params3 := PriceParams{
		IDs:        []string{"ethereum"},
		Currencies: []string{"usd"},
	}
	keys3 := createCacheKeys(params3)
	assert.Len(t, keys3, 1)
	assert.Equal(t, "simple_price:ethereum", keys3[0])
	assert.NotEqual(t, keys1[0], keys3[0])

	// Test multiple tokens
	params4 := PriceParams{
		IDs:        []string{"bitcoin", "ethereum"},
		Currencies: []string{"usd"},
	}
	keys4 := createCacheKeys(params4)
	assert.Len(t, keys4, 2)
	assert.Equal(t, "simple_price:bitcoin", keys4[0])
	assert.Equal(t, "simple_price:ethereum", keys4[1])

	// All keys should contain the prefix
	for _, key := range keys4 {
		assert.Contains(t, key, "simple_price:")
	}
}

func TestExtractTokenIDFromKey(t *testing.T) {
	// Test valid cache key
	key := "simple_price:bitcoin"
	tokenID := extractTokenIDFromKey(key)
	assert.Equal(t, "bitcoin", tokenID)

	// Test another valid key
	key2 := "simple_price:ethereum"
	tokenID2 := extractTokenIDFromKey(key2)
	assert.Equal(t, "ethereum", tokenID2)

	// Test invalid key
	invalidKey := "invalid:key"
	tokenID3 := extractTokenIDFromKey(invalidKey)
	assert.Equal(t, "key", tokenID3)

	// Test empty key
	emptyKey := ""
	tokenID4 := extractTokenIDFromKey(emptyKey)
	assert.Equal(t, "", tokenID4)
}

func TestExtractTokensFromKeys(t *testing.T) {
	keys := []string{
		"simple_price:bitcoin",
		"simple_price:ethereum",
		"simple_price:bitcoin", // Duplicate bitcoin
		"simple_price:cardano",
	}

	tokens := extractTokensFromKeys(keys)

	// Should have unique tokens only
	assert.Len(t, tokens, 3)
	assert.Contains(t, tokens, "bitcoin")
	assert.Contains(t, tokens, "ethereum")
	assert.Contains(t, tokens, "cardano")

	// Test empty keys
	emptyResult := extractTokensFromKeys([]string{})
	assert.Len(t, emptyResult, 0)

	// Test invalid keys
	invalidKeys := []string{"invalid", "also:invalid"}
	invalidResult := extractTokensFromKeys(invalidKeys)
	assert.Len(t, invalidResult, 1) // Only "invalid" from "also:invalid" should remain
	assert.Contains(t, invalidResult, "invalid")
}
