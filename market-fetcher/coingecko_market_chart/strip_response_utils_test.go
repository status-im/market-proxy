package coingecko_market_chart

import (
	"encoding/json"
	"testing"
	"time"
)

// Helper function to create test market chart data
func createTestMarketChartData(days int) []byte {
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

	response := MarketChartResponse{
		Prices:       prices,
		MarketCaps:   marketCaps,
		TotalVolumes: totalVolumes,
	}

	data, _ := json.Marshal(response)
	return data
}

// Helper function to create test response map
func createTestResponseMap(days int) map[string][]byte {
	return map[string][]byte{
		"prices":        createTestMarketChartData(days),
		"market_caps":   createTestMarketChartData(days),
		"total_volumes": createTestMarketChartData(days),
	}
}

func TestStripMarketChartResponse_MaxDays(t *testing.T) {
	// Test that "max" days returns all data unchanged
	originalParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "max",
	}

	enrichedResponse := createTestResponseMap(90)
	result, err := StripMarketChartResponse(originalParams, enrichedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(result) != len(enrichedResponse) {
		t.Errorf("Expected result length %d, got %d", len(enrichedResponse), len(result))
	}

	// Verify data is unchanged
	for key, originalData := range enrichedResponse {
		if resultData, exists := result[key]; !exists {
			t.Errorf("Expected key %s to exist in result", key)
		} else if string(resultData) != string(originalData) {
			t.Errorf("Expected data for key %s to be unchanged", key)
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

	enrichedResponse := createTestResponseMap(90)
	result, err := StripMarketChartResponse(originalParams, enrichedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(result) != len(enrichedResponse) {
		t.Errorf("Expected result length %d, got %d", len(enrichedResponse), len(result))
	}

	// Verify that data was filtered
	for key, resultData := range result {
		var chartResponse MarketChartResponse
		if err := json.Unmarshal(resultData, &chartResponse); err != nil {
			t.Errorf("Failed to unmarshal result data for key %s: %v", key, err)
			continue
		}

		// Check that we have fewer data points than the original 90 days
		if len(chartResponse.Prices) >= 90 {
			t.Errorf("Expected filtered data to have fewer than 90 points, got %d", len(chartResponse.Prices))
		}

		// Check that we have at least some data points
		if len(chartResponse.Prices) == 0 {
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

	enrichedResponse := createTestResponseMap(30)
	result, err := StripMarketChartResponse(originalParams, enrichedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should return all data unchanged
	if len(result) != len(enrichedResponse) {
		t.Errorf("Expected result length %d, got %d", len(enrichedResponse), len(result))
	}
}

func TestStripMarketChartResponse_InvalidDays(t *testing.T) {
	// Test with invalid days parameter
	originalParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "invalid",
	}

	enrichedResponse := createTestResponseMap(30)
	result, err := StripMarketChartResponse(originalParams, enrichedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should return all data unchanged when days parsing fails
	if len(result) != len(enrichedResponse) {
		t.Errorf("Expected result length %d, got %d", len(enrichedResponse), len(result))
	}
}

func TestStripMarketChartResponse_InvalidJSON(t *testing.T) {
	// Test with invalid JSON data
	originalParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "7",
	}

	enrichedResponse := map[string][]byte{
		"prices": []byte("invalid json"),
	}

	result, err := StripMarketChartResponse(originalParams, enrichedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should return original data when JSON parsing fails
	if len(result) != len(enrichedResponse) {
		t.Errorf("Expected result length %d, got %d", len(enrichedResponse), len(result))
	}

	if string(result["prices"]) != string(enrichedResponse["prices"]) {
		t.Errorf("Expected original data to be preserved when JSON parsing fails")
	}
}

func TestStripMarketChartResponseInPlace(t *testing.T) {
	// Test in-place modification
	originalParams := MarketChartParams{
		ID:       "bitcoin",
		Currency: "usd",
		Days:     "7",
	}

	enrichedResponse := createTestResponseMap(30)
	originalData := make(map[string][]byte)
	for k, v := range enrichedResponse {
		originalData[k] = make([]byte, len(v))
		copy(originalData[k], v)
	}

	err := StripMarketChartResponseInPlace(originalParams, enrichedResponse)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify that original map was modified
	dataChanged := false
	for key, newData := range enrichedResponse {
		if string(newData) != string(originalData[key]) {
			dataChanged = true
			break
		}
	}

	if !dataChanged {
		t.Error("Expected data to be modified in place")
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
