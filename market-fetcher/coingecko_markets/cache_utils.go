package coingecko_markets

import (
	"encoding/json"
	"fmt"
)

// parseTokensData parses tokens data and extracts market data with cache keys
// Skips invalid data entries without causing errors (graceful handling for API responses)
func parseTokensData(tokensData [][]byte) ([]interface{}, map[string][]byte, error) {
	marketData := make([]interface{}, 0, len(tokensData))
	cacheData := make(map[string][]byte)

	for _, tokenBytes := range tokensData {
		var tokenData interface{}
		if err := json.Unmarshal(tokenBytes, &tokenData); err != nil {
			// Skip malformed JSON data - this can happen with real API responses
			continue
		}

		// Extract ID and create cache key directly
		if tokenMap, ok := tokenData.(map[string]interface{}); ok {
			if id, exists := tokenMap[ID_FIELD]; exists {
				if tokenID, ok := id.(string); ok && tokenID != "" {
					cacheKey := getCacheKey(tokenID)
					cacheData[cacheKey] = tokenBytes
					marketData = append(marketData, tokenData)
				}
			}
		}
	}

	return marketData, cacheData, nil
}

// parsePagesData parses pages data and extracts page mapping with cache keys
func parsePagesData(pagesData []PageData) (map[int]interface{}, map[string][]byte, error) {
	pageMapping := make(map[int]interface{})
	cacheData := make(map[string][]byte)

	for _, pageData := range pagesData {
		// Create page cache key for full page data
		cacheKey := createPageCacheKey(pageData.Page)

		// Serialize page data
		pageBytes, err := json.Marshal(pageData.Data)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal page data for page %d: %w", pageData.Page, err)
		}

		cacheData[cacheKey] = pageBytes

		// Add page data to mapping
		pageMapping[pageData.Page] = pageData.Data

		// Extract token IDs from page data and create page IDs cache entry
		tokenIDs := extractTokenIDsFromPageData(pageData.Data)
		if len(tokenIDs) > 0 {
			pageIdsCacheKey := createPageIdsCacheKey(pageData.Page)
			tokenIDsBytes, err := json.Marshal(tokenIDs)
			if err == nil {
				cacheData[pageIdsCacheKey] = tokenIDsBytes
			}
		}
	}

	return pageMapping, cacheData, nil
}

// extractTokenIDsFromPageData extracts token IDs from page data preserving order
func extractTokenIDsFromPageData(pageData [][]byte) []string {
	var tokenIDs []string

	for _, tokenBytes := range pageData {
		var tokenData interface{}
		if err := json.Unmarshal(tokenBytes, &tokenData); err != nil {
			continue
		}

		// Extract ID from token data
		if tokenMap, ok := tokenData.(map[string]interface{}); ok {
			if id, exists := tokenMap[ID_FIELD]; exists {
				if tokenID, ok := id.(string); ok && tokenID != "" {
					tokenIDs = append(tokenIDs, tokenID)
				}
			}
		}
	}

	return tokenIDs
}
