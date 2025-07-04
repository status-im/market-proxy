package coingecko_market_chart

import (
	"encoding/json"
	"log"
	"strconv"
	"time"
)

// StripMarketChartResponse filters the enriched market chart response data
// to match the original request parameters. This is used after fetching
// enriched data (e.g., 90 days instead of 30) to return only the data
// that was originally requested.
func StripMarketChartResponse(originalParams MarketChartParams, enrichedResponse map[string][]byte) (map[string][]byte, error) {
	// If the original request was for "max" days, return all data
	if originalParams.Days == "max" {
		return enrichedResponse, nil
	}

	// If the original request was for specific days, filter by days
	if originalParams.Days != "" {
		return filterByDays(enrichedResponse, originalParams.Days)
	}

	// If no filtering criteria, return all data
	return enrichedResponse, nil
}

// filterByDays filters the response data to include only the last N days
func filterByDays(responseData map[string][]byte, daysStr string) (map[string][]byte, error) {
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		log.Printf("StripMarketChartResponse: Unable to parse days '%s' as integer, returning all data", daysStr)
		return responseData, nil
	}

	// Calculate the timestamp for N days ago
	cutoffTime := time.Now().AddDate(0, 0, -days).Unix() * 1000 // Convert to milliseconds

	result := make(map[string][]byte)
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
func filterChartDataByTimestamp(data []byte, cutoffTimestamp int64) ([]byte, error) {
	var chartResponse MarketChartResponse
	if err := json.Unmarshal(data, &chartResponse); err != nil {
		return nil, err
	}

	// Filter each data array
	chartResponse.Prices = filterDataPoints(chartResponse.Prices, cutoffTimestamp, 0)
	chartResponse.MarketCaps = filterDataPoints(chartResponse.MarketCaps, cutoffTimestamp, 0)
	chartResponse.TotalVolumes = filterDataPoints(chartResponse.TotalVolumes, cutoffTimestamp, 0)

	// Marshal back to JSON
	return json.Marshal(chartResponse)
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

// StripMarketChartResponseInPlace strips the enriched response data in place
// This is a convenience function that modifies the original response map
func StripMarketChartResponseInPlace(originalParams MarketChartParams, enrichedResponse map[string][]byte) error {
	stripped, err := StripMarketChartResponse(originalParams, enrichedResponse)
	if err != nil {
		return err
	}

	// Replace the original data with stripped data
	for key, data := range stripped {
		enrichedResponse[key] = data
	}

	return nil
}
