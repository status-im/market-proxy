package coingecko_prices

import (
	"fmt"
	"strings"
)

// createCacheKeys creates cache keys for each token ID
func createCacheKeys(params PriceParams) []string {
	keys := make([]string, len(params.IDs))

	// Create a key for each token ID (without currencies)
	for i, tokenID := range params.IDs {
		keys[i] = fmt.Sprintf("simple_price:%s", tokenID)
	}

	return keys
}

// extractTokensFromKeys extracts unique token IDs from cache keys
func extractTokensFromKeys(keys []string) []string {
	tokenSet := make(map[string]bool)

	for _, key := range keys {
		tokenID := extractTokenIDFromKey(key)
		if tokenID != "" {
			tokenSet[tokenID] = true
		}
	}

	tokens := make([]string, 0, len(tokenSet))
	for token := range tokenSet {
		tokens = append(tokens, token)
	}

	return tokens
}

// extractTokenIDFromKey extracts token ID from cache key
// Cache key format: "simple_price:{tokenID}"
func extractTokenIDFromKey(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
