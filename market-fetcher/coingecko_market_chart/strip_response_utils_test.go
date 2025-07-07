package coingecko_market_chart

import (
	"testing"
	"time"
)

// Helper function to create test response map
func createTestResponseMap(days int) map[string]interface{} {
	// Create test data
	now := time.Now()
	var prices []MarketChartData
	var marketCaps []MarketChartData
	var totalVolumes []MarketChartData

	// Create data for the specified number of days
	for i := 0; i < days; i++ {
		timestamp := now.AddDate(0, 0, -days+i).Unix() * 1000 // milliseconds
		price := float64(50000 + i*100)                       // Mock price data
		marketCap := float64(1000000000 + i*1000000)          // Mock market cap data
		volume := float64(10000000 + i*100000)                // Mock volume data

		prices = append(prices, MarketChartData{float64(timestamp), price})
		marketCaps = append(marketCaps, MarketChartData{float64(timestamp), marketCap})
		totalVolumes = append(totalVolumes, MarketChartData{float64(timestamp), volume})
	}

	return map[string]interface{}{
		"prices":        prices,
		"market_caps":   marketCaps,
		"total_volumes": totalVolumes,
	}
}

func TestStripMarketChartResponse_MaxDays(t *testing.T) {
	// Test that "max" days returns all data unchanged
	originalParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "max",
	}

	roundedResponse := createTestResponseMap(90)
	result, err := StripMarketChartResponse(originalParams, roundedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(result) != len(roundedResponse) {
		t.Errorf("Expected result length %d, got %d", len(roundedResponse), len(result))
	}

	// Verify data is unchanged
	for key := range roundedResponse {
		if _, exists := result[key]; !exists {
			t.Errorf("Expected key %s to exist in result", key)
		}
	}
}

func TestStripMarketChartResponse_FilterByDays(t *testing.T) {
	// Test filtering 90 days of data to 7 days
	originalParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "7",
	}

	roundedResponse := createTestResponseMap(90)
	result, err := StripMarketChartResponse(originalParams, roundedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(result) != len(roundedResponse) {
		t.Errorf("Expected result length %d, got %d", len(roundedResponse), len(result))
	}

	// Verify that data was filtered
	for key, resultData := range result {
		// Convert interface{} to []MarketChartData for verification
		prices, ok := resultData.([]MarketChartData)
		if !ok {
			t.Errorf("Expected data for key %s to be []MarketChartData", key)
			continue
		}

		// Check that we have fewer data points than the original 90 days
		if len(prices) >= 90 {
			t.Errorf("Expected filtered data to have fewer than 90 points, got %d", len(prices))
		}

		// Check that we have at least some data points
		if len(prices) == 0 {
			t.Errorf("Expected filtered data to have some points, got 0")
		}
	}
}

func TestStripMarketChartResponse_EmptyDays(t *testing.T) {
	// Test with empty days parameter
	originalParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "",
	}

	roundedResponse := createTestResponseMap(30)
	result, err := StripMarketChartResponse(originalParams, roundedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should return all data unchanged
	if len(result) != len(roundedResponse) {
		t.Errorf("Expected result length %d, got %d", len(roundedResponse), len(result))
	}
}

func TestStripMarketChartResponse_InvalidDays(t *testing.T) {
	// Test with invalid days parameter
	originalParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "invalid",
	}

	roundedResponse := createTestResponseMap(30)
	result, err := StripMarketChartResponse(originalParams, roundedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should return all data unchanged when days parsing fails
	if len(result) != len(roundedResponse) {
		t.Errorf("Expected result length %d, got %d", len(roundedResponse), len(result))
	}
}

func TestStripMarketChartResponse_InvalidJSON(t *testing.T) {
	// Test with invalid JSON data
	originalParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "7",
	}

	roundedResponse := map[string]interface{}{
		"prices": "invalid json",
	}

	result, err := StripMarketChartResponse(originalParams, roundedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should return original data when JSON parsing fails
	if len(result) != len(roundedResponse) {
		t.Errorf("Expected result length %d, got %d", len(roundedResponse), len(result))
	}

	if result["prices"] != roundedResponse["prices"] {
		t.Errorf("Expected original data to be preserved when JSON parsing fails")
	}
}

func TestFilterDataPoints(t *testing.T) {
	// Test the filterDataPoints helper function
	now := time.Now()
	cutoffTime := now.AddDate(0, 0, -7).Unix() * 1000 // 7 days ago in milliseconds

	// Create test data spanning 14 days
	var dataPoints []MarketChartData
	for i := 0; i < 14; i++ {
		timestamp := now.AddDate(0, 0, -14+i).Unix() * 1000
		value := float64(i * 100)
		dataPoints = append(dataPoints, MarketChartData{float64(timestamp), value})
	}

	filtered := filterDataPoints(dataPoints, cutoffTime, 0)

	// Should have approximately 7 data points (or fewer due to time precision)
	if len(filtered) > 8 {
		t.Errorf("Expected filtered data to have around 7 points, got %d", len(filtered))
	}

	// All filtered points should be after cutoff time
	for _, point := range filtered {
		if point[0] < float64(cutoffTime) {
			t.Errorf("Found point with timestamp %f before cutoff %d", point[0], cutoffTime)
		}
	}
}

func TestStripMarketChartResponse_FilterByDataKeys(t *testing.T) {
	// Test filtering by data keys
	originalParams := MarketChartParams{
		ID:         "bitcoin",
		Currency:   "usd",
		Days:       "max",
		DataFilter: "prices,market_caps",
	}

	roundedResponse := createTestResponseMap(30)
	result, err := StripMarketChartResponse(originalParams, roundedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should have only 2 keys: prices and market_caps
	if len(result) != 2 {
		t.Errorf("Expected result length 2, got %d", len(result))
	}

	// Check that prices and market_caps are present
	if _, exists := result["prices"]; !exists {
		t.Error("Expected 'prices' key to exist in result")
	}
	if _, exists := result["market_caps"]; !exists {
		t.Error("Expected 'market_caps' key to exist in result")
	}

	// Check that total_volumes is not present
	if _, exists := result["total_volumes"]; exists {
		t.Error("Expected 'total_volumes' key to be filtered out")
	}
}

func TestStripMarketChartResponse_FilterByDataKeysAndDays(t *testing.T) {
	// Test filtering by both data keys and days
	originalParams := MarketChartParams{
		ID:         "bitcoin",
		Currency:   "usd",
		Days:       "7",
		DataFilter: "prices",
	}

	roundedResponse := createTestResponseMap(90)
	result, err := StripMarketChartResponse(originalParams, roundedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should have only 1 key: prices
	if len(result) != 1 {
		t.Errorf("Expected result length 1, got %d", len(result))
	}

	// Check that only prices is present
	if _, exists := result["prices"]; !exists {
		t.Error("Expected 'prices' key to exist in result")
	}

	// Verify that data was also filtered by days
	prices, ok := result["prices"].([]MarketChartData)
	if !ok {
		t.Error("Expected 'prices' to be []MarketChartData")
	} else {
		// Check that we have fewer data points than the original 90 days
		if len(prices) >= 90 {
			t.Errorf("Expected filtered data to have fewer than 90 points, got %d", len(prices))
		}
	}
}

func TestStripMarketChartResponse_EmptyDataFilter(t *testing.T) {
	// Test with empty data filter - should include all keys
	originalParams := MarketChartParams{
		ID:         "bitcoin",
		Currency:   "usd",
		Days:       "max",
		DataFilter: "",
	}

	roundedResponse := createTestResponseMap(30)
	result, err := StripMarketChartResponse(originalParams, roundedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should have all 3 keys
	if len(result) != 3 {
		t.Errorf("Expected result length 3, got %d", len(result))
	}

	// Check that all keys are present
	expectedKeys := []string{"prices", "market_caps", "total_volumes"}
	for _, key := range expectedKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("Expected '%s' key to exist in result", key)
		}
	}
}

func TestStripMarketChartResponse_DataFilterWithSpaces(t *testing.T) {
	// Test data filter with spaces around commas
	originalParams := MarketChartParams{
		ID:         "bitcoin",
		Currency:   "usd",
		Days:       "max",
		DataFilter: " prices , total_volumes ",
	}

	roundedResponse := createTestResponseMap(30)
	result, err := StripMarketChartResponse(originalParams, roundedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should have only 2 keys: prices and total_volumes
	if len(result) != 2 {
		t.Errorf("Expected result length 2, got %d", len(result))
	}

	// Check that prices and total_volumes are present
	if _, exists := result["prices"]; !exists {
		t.Error("Expected 'prices' key to exist in result")
	}
	if _, exists := result["total_volumes"]; !exists {
		t.Error("Expected 'total_volumes' key to exist in result")
	}

	// Check that market_caps is not present
	if _, exists := result["market_caps"]; exists {
		t.Error("Expected 'market_caps' key to be filtered out")
	}
}

func TestFilterByDataKeys(t *testing.T) {
	// Test the filterByDataKeys function directly
	responseData := map[string]interface{}{
		"prices":        []MarketChartData{{123456789, 50000}},
		"market_caps":   []MarketChartData{{123456789, 1000000000}},
		"total_volumes": []MarketChartData{{123456789, 10000000}},
	}

	tests := []struct {
		name           string
		dataFilter     string
		expectedKeys   []string
		unexpectedKeys []string
	}{
		{
			name:           "Filter prices only",
			dataFilter:     "prices",
			expectedKeys:   []string{"prices"},
			unexpectedKeys: []string{"market_caps", "total_volumes"},
		},
		{
			name:           "Filter prices and market_caps",
			dataFilter:     "prices,market_caps",
			expectedKeys:   []string{"prices", "market_caps"},
			unexpectedKeys: []string{"total_volumes"},
		},
		{
			name:           "Empty filter",
			dataFilter:     "",
			expectedKeys:   []string{"prices", "market_caps", "total_volumes"},
			unexpectedKeys: []string{},
		},
		{
			name:           "Filter with spaces",
			dataFilter:     " prices , total_volumes ",
			expectedKeys:   []string{"prices", "total_volumes"},
			unexpectedKeys: []string{"market_caps"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterByDataKeys(responseData, tt.dataFilter)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			// Check expected keys
			for _, key := range tt.expectedKeys {
				if _, exists := result[key]; !exists {
					t.Errorf("Expected key '%s' to exist in result", key)
				}
			}

			// Check unexpected keys
			for _, key := range tt.unexpectedKeys {
				if _, exists := result[key]; exists {
					t.Errorf("Expected key '%s' to be filtered out", key)
				}
			}

			// Check total count
			if len(result) != len(tt.expectedKeys) {
				t.Errorf("Expected result length %d, got %d", len(tt.expectedKeys), len(result))
			}
		})
	}
}
