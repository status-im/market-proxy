package coingecko_market_chart

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
)

// StripMarketChartResponse filters rounded response data to match original request parameters
func StripMarketChartResponse(originalParams MarketChartParams, roundedResponse map[string]interface{}) (map[string]interface{}, error) {
	result := roundedResponse
	if originalParams.DataFilter != "" {
		var err error
		result, err = filterByDataKeys(roundedResponse, originalParams.DataFilter)
		if err != nil {
			return nil, err
		}
	}

	if originalParams.Days == "max" {
		return result, nil
	}

	if originalParams.Days != "" {
		return filterByDays(result, originalParams.Days)
	}

	return result, nil
}

func filterByDataKeys(responseData map[string]interface{}, dataFilter string) (map[string]interface{}, error) {
	if dataFilter == "" {
		return responseData, nil
	}

	allowedKeys := make(map[string]bool)
	keys := strings.Split(dataFilter, ",")
	for _, key := range keys {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey != "" {
			allowedKeys[trimmedKey] = true
		}
	}

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

func filterByDays(responseData map[string]interface{}, daysStr string) (map[string]interface{}, error) {
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		log.Printf("StripMarketChartResponse: Unable to parse days '%s' as integer, returning all data", daysStr)
		return responseData, nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -days).Unix() * 1000

	result := make(map[string]interface{})
	for key, data := range responseData {
		filteredData, err := filterChartDataByTimestamp(data, cutoffTime)
		if err != nil {
			log.Printf("StripMarketChartResponse: Error filtering data for key %s: %v", key, err)
			result[key] = data
		} else {
			result[key] = filteredData
		}
	}

	return result, nil
}

func filterChartDataByTimestamp(data interface{}, cutoffTimestamp int64) (interface{}, error) {
	var dataPoints []MarketChartData

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(dataBytes, &dataPoints); err != nil {
		return nil, err
	}

	filteredPoints := filterDataPoints(dataPoints, cutoffTimestamp, 0)

	return filteredPoints, nil
}

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

	if minPoints > 0 && len(filtered) < minPoints && len(dataPoints) > 0 {
		start := len(dataPoints) - minPoints
		if start < 0 {
			start = 0
		}
		filtered = dataPoints[start:]
	}

	return filtered
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
