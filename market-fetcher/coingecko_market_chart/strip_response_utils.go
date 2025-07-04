package coingecko_market_chart

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
)

// StripMarketChartResponse filters the enriched market chart response data
// to match the original request parameters. This is used after fetching
// enriched data (e.g., 90 days instead of 30) to return only the data
// that was originally requested.
func StripMarketChartResponse(originalParams MarketChartParams, enrichedResponse map[string]interface{}) (map[string]interface{}, error) {
	// Step 1: Filter by data keys if DataFilter is specified
	result := enrichedResponse
	if originalParams.DataFilter != "" {
		var err error
		result, err = filterByDataKeys(enrichedResponse, originalParams.DataFilter)
		if err != nil {
			return nil, err
		}
	}

	// Step 2: Filter by days
	// If the original request was for "max" days, return data as is
	if originalParams.Days == "max" {
		return result, nil
	}

	// If the original request was for specific days, filter by days
	if originalParams.Days != "" {
		return filterByDays(result, originalParams.Days)
	}

	// If no filtering criteria, return all data
	return result, nil
}

// filterByDataKeys filters the response data to include only specified keys
func filterByDataKeys(responseData map[string]interface{}, dataFilter string) (map[string]interface{}, error) {
	if dataFilter == "" {
		return responseData, nil
	}

	// Parse the comma-separated list of keys
	allowedKeys := make(map[string]bool)
	keys := strings.Split(dataFilter, ",")
	for _, key := range keys {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey != "" {
			allowedKeys[trimmedKey] = true
		}
	}

	// Filter the response data
	result := make(map[string]interface{})
	for key, data := range responseData {
		if allowedKeys[key] {
			result[key] = data
		}
	}

	log.Printf("StripMarketChartResponse: Filtered data keys from %v to %v based on filter '%s'",
		getKeys(responseData), getKeys(result), dataFilter)

	return result, nil
}

// filterByDays filters the response data to include only the last N days
func filterByDays(responseData map[string]interface{}, daysStr string) (map[string]interface{}, error) {
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		log.Printf("StripMarketChartResponse: Unable to parse days '%s' as integer, returning all data", daysStr)
		return responseData, nil
	}

	// Calculate the timestamp for N days ago
	cutoffTime := time.Now().AddDate(0, 0, -days).Unix() * 1000 // Convert to milliseconds

	result := make(map[string]interface{})
	for key, data := range responseData {
		filteredData, err := filterChartDataByTimestamp(data, cutoffTime)
		if err != nil {
			log.Printf("StripMarketChartResponse: Error filtering data for key %s: %v", key, err)
			// If filtering fails, include the original data
			result[key] = data
		} else {
			result[key] = filteredData
		}
	}

	return result, nil
}

// filterChartDataByTimestamp filters chart data to include only entries after the cutoff timestamp
func filterChartDataByTimestamp(data interface{}, cutoffTimestamp int64) (interface{}, error) {
	// Convert interface{} to []MarketChartData
	var dataPoints []MarketChartData

	// First, convert to JSON bytes, then unmarshal
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(dataBytes, &dataPoints); err != nil {
		return nil, err
	}

	// Filter the data points
	filteredPoints := filterDataPoints(dataPoints, cutoffTimestamp, 0)

	// Return as interface{} (will be []MarketChartData)
	return filteredPoints, nil
}

// filterDataPoints filters data points to include only those after the cutoff timestamp
func filterDataPoints(dataPoints []MarketChartData, cutoffTimestamp int64, minPoints int) []MarketChartData {
	if len(dataPoints) == 0 {
		return dataPoints
	}

	var filtered []MarketChartData
	for _, point := range dataPoints {
		if point[0] >= float64(cutoffTimestamp) {
			filtered = append(filtered, point)
		}
	}

	// Ensure we have at least minPoints (if specified)
	if minPoints > 0 && len(filtered) < minPoints && len(dataPoints) > 0 {
		// Take the last minPoints from the original data
		start := len(dataPoints) - minPoints
		if start < 0 {
			start = 0
		}
		filtered = dataPoints[start:]
	}

	return filtered
}

// getKeys returns a slice of keys from a map for logging purposes
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// StripMarketChartResponseInPlace strips the enriched response data in place
// This is a convenience function that modifies the original response map
func StripMarketChartResponseInPlace(originalParams MarketChartParams, enrichedResponse map[string]interface{}) error {
	stripped, err := StripMarketChartResponse(originalParams, enrichedResponse)
	if err != nil {
		return err
	}

	// Clear the original map and replace with stripped data
	for key := range enrichedResponse {
		delete(enrichedResponse, key)
	}
	for key, data := range stripped {
		enrichedResponse[key] = data
	}

	return nil
}
