package coingecko_prices

import (
	"math"
	"strconv"
	"strings"

	cg "github.com/status-im/market-proxy/interfaces"
)

// stripResponse filters the cached response to only include user-requested currencies and fields
func stripResponse(cachedResponse cg.SimplePriceResponse, params cg.PriceParams) cg.SimplePriceResponse {
	result := make(cg.SimplePriceResponse)

	// Create set of requested currencies for fast lookup
	requestedCurrencies := make(map[string]bool)
	for _, currency := range params.Currencies {
		requestedCurrencies[currency] = true
	}

	// Parse precision from string
	precision := 0
	if params.Precision != "" {
		if p, err := strconv.Atoi(params.Precision); err == nil && p > 0 {
			precision = p
		}
	}

	// Process each token in the cached response
	for tokenID, tokenData := range cachedResponse {
		if tokenMap, ok := tokenData.(map[string]interface{}); ok {
			filteredTokenData := make(map[string]interface{})

			// Filter fields based on requested currencies and include flags
			for fieldName, value := range tokenMap {
				if shouldIncludeField(fieldName, requestedCurrencies, params) {
					// Apply precision if it's a numeric value and precision is specified
					if precision > 0 && isNumericField(fieldName) {
						if numValue, ok := value.(float64); ok {
							value = roundToPrecision(numValue, precision)
						}
					}
					filteredTokenData[fieldName] = value
				}
			}

			// Add filtered token data to result if it has any requested fields
			if len(filteredTokenData) > 0 {
				result[tokenID] = filteredTokenData
			}
		}
	}

	return result
}

// shouldIncludeField determines if a field should be included based on user parameters
func shouldIncludeField(fieldName string, requestedCurrencies map[string]bool, params cg.PriceParams) bool {
	// Check if it's a base currency field (e.g., "usd", "eur", "btc")
	if requestedCurrencies[fieldName] {
		return true
	}

	// Check if it's a market cap field (e.g., "usd_market_cap", "btc_market_cap")
	if params.IncludeMarketCap && strings.HasSuffix(fieldName, "_market_cap") {
		currency := strings.TrimSuffix(fieldName, "_market_cap")
		return requestedCurrencies[currency]
	}

	// Check if it's a 24h volume field (e.g., "usd_24h_vol", "btc_24h_vol")
	if params.Include24hrVol && strings.HasSuffix(fieldName, "_24h_vol") {
		currency := strings.TrimSuffix(fieldName, "_24h_vol")
		return requestedCurrencies[currency]
	}

	// Check if it's a 24h change field (e.g., "usd_24h_change", "btc_24h_change")
	if params.Include24hrChange && strings.HasSuffix(fieldName, "_24h_change") {
		currency := strings.TrimSuffix(fieldName, "_24h_change")
		return requestedCurrencies[currency]
	}

	// Check if it's the last updated field
	if params.IncludeLastUpdatedAt && fieldName == "last_updated_at" {
		return true
	}

	return false
}

// isNumericField checks if a field contains numeric data that should be subject to precision rounding
func isNumericField(fieldName string) bool {
	// Don't round last_updated_at as it's a timestamp
	if fieldName == "last_updated_at" {
		return false
	}
	// All other fields (prices, market_cap, volume, change) should be rounded
	return true
}

// roundToPrecision rounds a number to the specified number of decimal places
func roundToPrecision(value float64, precision int) float64 {
	if precision < 0 {
		return value // negative precision returns original
	}

	if precision == 0 {
		return math.Round(value) // round to nearest integer
	}

	multiplier := math.Pow(10, float64(precision))
	return math.Round(value*multiplier) / multiplier
}
