package coingecko_markets

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/status-im/market-proxy/interfaces"
)

const (
	// CACHE_KEY_PREFIX is the prefix used for cache keys in markets module
	CACHE_KEY_PREFIX = "markets:"

	// CACHE_KEY_PAGE_PREFIX is the prefix used for page-based cache keys in markets module
	CACHE_KEY_PAGE_PREFIX = "markets_page:"
)

// createCacheKeys creates cache keys for each token ID in MarketParams
func createCacheKeys(params interfaces.MarketsParams) []string {
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

// createPageCacheKey creates a single cache key for page-based requests
func createPageCacheKey(pageID int) string {
	return fmt.Sprintf("%s%d", CACHE_KEY_PAGE_PREFIX, pageID)
}

// ConvertMarketsResponseToCoinGeckoData converts raw markets response data to CoinGeckoData slice
// This function processes the [][]byte from coins/markets API, unmarshals each item, and converts to CoinGeckoData
func ConvertMarketsResponseToCoinGeckoData(tokensData [][]byte) []CoinGeckoData {
	// Convert raw markets data to []interface{}
	marketsData := make([]interface{}, 0, len(tokensData))
	for _, tokenBytes := range tokensData {
		var tokenData interface{}
		if err := json.Unmarshal(tokenBytes, &tokenData); err == nil {
			marketsData = append(marketsData, tokenData)
		}
	}

	result := make([]CoinGeckoData, 0, len(marketsData))

	for _, item := range marketsData {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Convert map[string]interface{} to CoinGeckoData directly
		coinData := CoinGeckoData{
			ID:                           getStringFromMap(itemMap, "id"),
			Symbol:                       getStringFromMap(itemMap, "symbol"),
			Name:                         getStringFromMap(itemMap, "name"),
			Image:                        getStringFromMap(itemMap, "image"),
			CurrentPrice:                 getFloatFromMap(itemMap, "current_price"),
			MarketCap:                    getFloatFromMap(itemMap, "market_cap"),
			MarketCapRank:                getIntFromMap(itemMap, "market_cap_rank"),
			FullyDilutedValuation:        getFloatFromMap(itemMap, "fully_diluted_valuation"),
			TotalVolume:                  getFloatFromMap(itemMap, "total_volume"),
			High24h:                      getFloatFromMap(itemMap, "high_24h"),
			Low24h:                       getFloatFromMap(itemMap, "low_24h"),
			PriceChange24h:               getFloatFromMap(itemMap, "price_change_24h"),
			PriceChangePercentage24h:     getFloatFromMap(itemMap, "price_change_percentage_24h"),
			MarketCapChange24h:           getFloatFromMap(itemMap, "market_cap_change_24h"),
			MarketCapChangePercentage24h: getFloatFromMap(itemMap, "market_cap_change_percentage_24h"),
			CirculatingSupply:            getFloatFromMap(itemMap, "circulating_supply"),
			TotalSupply:                  getFloatFromMap(itemMap, "total_supply"),
			MaxSupply:                    getFloatFromMap(itemMap, "max_supply"),
			ATH:                          getFloatFromMap(itemMap, "ath"),
			ATHChangePercentage:          getFloatFromMap(itemMap, "ath_change_percentage"),
			ATHDate:                      getStringFromMap(itemMap, "ath_date"),
			ATL:                          getFloatFromMap(itemMap, "atl"),
			ATLChangePercentage:          getFloatFromMap(itemMap, "atl_change_percentage"),
			ATLDate:                      getStringFromMap(itemMap, "atl_date"),
			ROI:                          itemMap["roi"], // Keep as interface{}
			LastUpdated:                  getStringFromMap(itemMap, "last_updated"),
		}

		result = append(result, coinData)
	}

	return result
}

// getStringFromMap safely extracts string from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// getFloatFromMap safely extracts float64 from map
func getFloatFromMap(m map[string]interface{}, key string) float64 {
	if value, exists := m[key]; exists {
		if f, ok := value.(float64); ok {
			return f
		}
	}
	return 0.0
}

// getIntFromMap safely extracts int from map
func getIntFromMap(m map[string]interface{}, key string) int {
	if value, exists := m[key]; exists {
		if i, ok := value.(float64); ok {
			return int(i)
		}
		if i, ok := value.(int); ok {
			return i
		}
	}
	return 0
}
