package coingecko_prices

import (
	"fmt"
	"strings"
)

// createCacheKeys creates cache keys for each token ID
func createCacheKeys(params PriceParams) []string {
	keys := make([]string, len(params.IDs))

	// Create currencies string for the key
	currenciesStr := ""
	for i, currency := range params.Currencies {
		if i > 0 {
			currenciesStr += ","
		}
		currenciesStr += currency
	}

	// Create a key for each token ID
	for i, tokenID := range params.IDs {
		keys[i] = fmt.Sprintf("simple_price:%s:%s", tokenID, currenciesStr)
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
// Cache key format: "simple_price:{tokenID}:{currencies}"
func extractTokenIDFromKey(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
