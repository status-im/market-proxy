package coingecko_markets

import (
	"fmt"
	"strings"

	cg "github.com/status-im/market-proxy/coingecko_common"
)

const (
	// CACHE_KEY_PREFIX is the prefix used for cache keys in markets module
	CACHE_KEY_PREFIX = "markets:"
)

// createCacheKeys creates cache keys for each token ID in MarketParams
func createCacheKeys(params cg.MarketsParams) []string {
	if len(params.IDs) == 0 {
		return []string{}
	}

	keys := make([]string, len(params.IDs))

	// Create a key for each token ID
	for i, tokenID := range params.IDs {
		keys[i] = getCacheKey(tokenID)
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
// Cache key format: "markets:{tokenID}"
func extractTokenIDFromKey(key string) string {
	if strings.HasPrefix(key, CACHE_KEY_PREFIX) {
		return key[len(CACHE_KEY_PREFIX):]
	}
	return ""
}

// getCacheKey creates a cache key for a single token ID
func getCacheKey(tokenID string) string {
	return fmt.Sprintf("%s%s", CACHE_KEY_PREFIX, tokenID)
}
