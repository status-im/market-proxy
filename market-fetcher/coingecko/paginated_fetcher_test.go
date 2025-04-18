package coingecko

import (
	"errors"
	"testing"
	"time"
)

// MockAPIClient is a mock implementation of the APIClient interface for testing
type MockAPIClient struct {
	// Define how many items to return per page
	itemsPerPage [][]CoinData
	// Define which pages should return errors
	errorPages map[int]error
	// Track which pages were requested
	requestedPages []int
	// Health status flag
	isHealthy bool
}

// FetchPage implements the APIClient.FetchPage method for mocking
func (m *MockAPIClient) FetchPage(page, limit int) ([]CoinData, error) {
	// Record this page request
	m.requestedPages = append(m.requestedPages, page)

	// Check if this page should return an error
	if err, exists := m.errorPages[page]; exists {
		return nil, err
	}

	// Check if we have data for this page
	if page <= len(m.itemsPerPage) && page > 0 {
		// Get predefined items for this page
		items := m.itemsPerPage[page-1]

		// Apply limit if needed
		if limit < len(items) {
			return items[:limit], nil
		}

		return items, nil
	}

	// Return empty slice for pages beyond what we have defined
	return []CoinData{}, nil
}

// Healthy implements the APIClient.Healthy method for mocking
func (m *MockAPIClient) Healthy() bool {
	return m.isHealthy // Use the isHealthy field
}

// TestPaginatedFetcher_SinglePage tests fetching a single page of data
func TestPaginatedFetcher_SinglePage(t *testing.T) {
	// Create mock data
	mockItems := []CoinData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
		{ID: "ripple", Symbol: "xrp", Name: "Ripple"},
	}

	// Create mock API client with one page of data
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{mockItems},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with total limit matching our mock data
	fetcher := NewPaginatedFetcher(mockClient, len(mockItems), 10, 0)

	// Call FetchData
	response, err := fetcher.FetchData()

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that we got the right number of items
	if len(response.Data) != len(mockItems) {
		t.Errorf("Expected %d items, got %d", len(mockItems), len(response.Data))
	}

	// Check that we requested exactly one page
	if len(mockClient.requestedPages) != 1 || mockClient.requestedPages[0] != 1 {
		t.Errorf("Expected one page request for page 1, got: %v", mockClient.requestedPages)
	}

	// Check the actual items
	for i, item := range response.Data {
		if item.ID != mockItems[i].ID {
			t.Errorf("Expected item %d to be %s, got %s", i, mockItems[i].ID, item.ID)
		}
	}
}

// TestPaginatedFetcher_MultiPage tests fetching multiple pages of data
func TestPaginatedFetcher_MultiPage(t *testing.T) {
	// Create mock data for two pages
	page1Items := []CoinData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}

	page2Items := []CoinData{
		{ID: "ripple", Symbol: "xrp", Name: "Ripple"},
		{ID: "litecoin", Symbol: "ltc", Name: "Litecoin"},
	}

	page3Items := []CoinData{
		{ID: "cardano", Symbol: "ada", Name: "Cardano"},
	}

	// Create mock API client with multiple pages of data
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{page1Items, page2Items, page3Items},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with total limit requiring all pages, and minimal delay for tests
	totalItems := len(page1Items) + len(page2Items) + len(page3Items)
	fetcher := NewPaginatedFetcher(mockClient, totalItems, 2, 1) // Each page has limit 2, 1ms delay

	// Call FetchData
	response, err := fetcher.FetchData()

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that we got the right number of items
	if len(response.Data) != totalItems {
		t.Errorf("Expected %d items, got %d", totalItems, len(response.Data))
	}

	// Check that we requested all three pages
	if len(mockClient.requestedPages) != 3 {
		t.Errorf("Expected three page requests, got: %v", mockClient.requestedPages)
	}

	// Check the first item (should be from page 1)
	if response.Data[0].ID != "bitcoin" {
		t.Errorf("Expected first item to be bitcoin, got %s", response.Data[0].ID)
	}

	// Check the last item (should be from page 3)
	if response.Data[4].ID != "cardano" {
		t.Errorf("Expected last item to be cardano, got %s", response.Data[4].ID)
	}
}

// TestPaginatedFetcher_Limit tests that the fetcher respects the total limit
func TestPaginatedFetcher_Limit(t *testing.T) {
	// Create mock data with many items
	page1Items := []CoinData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
		{ID: "ripple", Symbol: "xrp", Name: "Ripple"},
	}

	page2Items := []CoinData{
		{ID: "litecoin", Symbol: "ltc", Name: "Litecoin"},
		{ID: "cardano", Symbol: "ada", Name: "Cardano"},
		{ID: "polkadot", Symbol: "dot", Name: "Polkadot"},
	}

	// Create mock API client with two pages of data
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{page1Items, page2Items},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with a limit less than the total available items
	limit := 4                                              // Less than the total 6 items
	fetcher := NewPaginatedFetcher(mockClient, limit, 3, 0) // Each page has limit 3, no delay

	// Call FetchData
	response, err := fetcher.FetchData()

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that we got exactly the limit number of items
	if len(response.Data) != limit {
		t.Errorf("Expected %d items (limit), got %d", limit, len(response.Data))
	}

	// Check that we requested both pages
	if len(mockClient.requestedPages) != 2 {
		t.Errorf("Expected two page requests, got: %v", mockClient.requestedPages)
	}

	// Check the first and last items to ensure they're correct
	if response.Data[0].ID != "bitcoin" {
		t.Errorf("Expected first item to be bitcoin, got %s", response.Data[0].ID)
	}
	if response.Data[3].ID != "litecoin" {
		t.Errorf("Expected fourth item to be litecoin, got %s", response.Data[3].ID)
	}
}

// TestPaginatedFetcher_ErrorFirstPage tests handling errors on the first page
func TestPaginatedFetcher_ErrorFirstPage(t *testing.T) {
	// Create mock API client with error on first page
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{},
		errorPages:   map[int]error{1: errors.New("API error on first page")},
	}

	// Create fetcher
	fetcher := NewPaginatedFetcher(mockClient, 10, 5, 0)

	// Call FetchData
	_, err := fetcher.FetchData()

	// Should get an error since the first page failed
	if err == nil {
		t.Error("Expected error when first page fails, got nil")
	}

	// Should have only tried the first page
	if len(mockClient.requestedPages) != 1 || mockClient.requestedPages[0] != 1 {
		t.Errorf("Expected one page request for page 1, got: %v", mockClient.requestedPages)
	}
}

func TestPaginatedFetcher_ErrorLaterPage(t *testing.T) {
	// Create mock data
	page1Items := []CoinData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}

	// Create mock API client with error on second page
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{page1Items},
		errorPages:   map[int]error{2: errors.New("API error on second page")},
	}

	// Create fetcher with total limit requiring multiple pages
	fetcher := NewPaginatedFetcher(mockClient, 5, 2, 0)

	// Call FetchData
	response, err := fetcher.FetchData()

	// Should not get an error since we got partial data
	if err != nil {
		t.Errorf("Expected no error with partial data, got: %v", err)
	}

	// Should get partial data (just from the first page)
	if len(response.Data) != len(page1Items) {
		t.Errorf("Expected %d items from first page, got %d", len(page1Items), len(response.Data))
	}

	// Check that we requested both pages
	if len(mockClient.requestedPages) != 2 {
		t.Errorf("Expected two page requests, got: %v", mockClient.requestedPages)
	}
}

// TestPaginatedFetcher_ZeroLimit tests that a zero limit request handles appropriately
func TestPaginatedFetcher_ZeroLimit(t *testing.T) {
	// Create mock API client
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with zero total limit
	fetcher := NewPaginatedFetcher(mockClient, 0, 10, 0)

	// Call FetchData
	response, err := fetcher.FetchData()

	// Should not get an error
	if err != nil {
		t.Errorf("Expected no error with zero limit, got: %v", err)
	}

	// Should get empty response
	if len(response.Data) != 0 {
		t.Errorf("Expected 0 items with zero limit, got %d", len(response.Data))
	}

	// Should not make any API requests
	if len(mockClient.requestedPages) != 0 {
		t.Errorf("Expected no page requests with zero limit, got: %v", mockClient.requestedPages)
	}
}

// TestPaginatedFetcher_LargeRequest tests fetching more data than actually available
func TestPaginatedFetcher_LargeRequest(t *testing.T) {
	// Create mock data with one page
	mockItems := []CoinData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}

	// Create mock API client with only one page of data
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{mockItems},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with a large limit
	limit := 100 // Much more than available
	fetcher := NewPaginatedFetcher(mockClient, limit, 10, 0)

	// Call FetchData
	response, err := fetcher.FetchData()

	// Should not get an error
	if err != nil {
		t.Fatalf("Expected no error with large request, got: %v", err)
	}

	// Should get only the available items
	if len(response.Data) != len(mockItems) {
		t.Errorf("Expected %d available items, got %d", len(mockItems), len(response.Data))
	}

	// Should have requested two pages (the second returning empty)
	if len(mockClient.requestedPages) < 2 {
		t.Errorf("Expected at least two page requests, got: %v", mockClient.requestedPages)
	}

	// The second page should be empty, triggering pagination stop
	lastPageRequested := mockClient.requestedPages[len(mockClient.requestedPages)-1]
	if lastPageRequested != 2 {
		t.Errorf("Expected pagination to stop after empty page 2, last page was: %d", lastPageRequested)
	}
}

// TestPaginatedFetcher_RequestDelay tests that the delay between requests is respected
func TestPaginatedFetcher_RequestDelay(t *testing.T) {
	// Skip in short mode as this test uses sleep
	if testing.Short() {
		t.Skip("Skipping delay test in short mode")
	}

	// Create mock data for multiple pages
	page1Items := []CoinData{{ID: "bitcoin"}}
	page2Items := []CoinData{{ID: "ethereum"}}
	page3Items := []CoinData{{ID: "ripple"}}

	// Create mock API client with multiple pages
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{page1Items, page2Items, page3Items},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with a significant delay (100ms for test)
	delay := 100 // 100ms delay between pages
	fetcher := NewPaginatedFetcher(mockClient, 3, 1, delay)

	// Record start time
	startTime := start()

	// Call FetchData
	_, err := fetcher.FetchData()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Calculate elapsed time
	elapsed := elapsed(startTime)

	// Check that the total time includes our delays
	// We expect at least 2 delays (between pages 1-2 and 2-3)
	minExpectedTime := time.Duration(delay*2) * time.Millisecond
	if elapsed < minExpectedTime {
		t.Errorf("Expected at least %v elapsed time due to delays, got %v",
			minExpectedTime, elapsed)
	}

	// Also verify we made requests for all pages
	if len(mockClient.requestedPages) != 3 {
		t.Errorf("Expected requests for 3 pages, got: %v", mockClient.requestedPages)
	}
}

// TestPaginatedFetcher_ZeroDelay tests that zero delay doesn't cause any waiting
func TestPaginatedFetcher_ZeroDelay(t *testing.T) {
	// Create mock data for multiple pages
	page1Items := []CoinData{{ID: "bitcoin"}}
	page2Items := []CoinData{{ID: "ethereum"}}
	page3Items := []CoinData{{ID: "ripple"}}

	// Create mock API client with multiple pages
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{page1Items, page2Items, page3Items},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with zero delay
	fetcher := NewPaginatedFetcher(mockClient, 3, 1, 0)

	// Record start time
	startTime := start()

	// Call FetchData
	_, err := fetcher.FetchData()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Calculate elapsed time
	elapsed := elapsed(startTime)

	// We expect the request to complete quickly with no delay
	// Allow a very small duration for test execution overhead
	maxExpectedTime := time.Duration(100) * time.Millisecond
	if elapsed > maxExpectedTime {
		t.Errorf("Expected fast execution with zero delay (< %v), got %v",
			maxExpectedTime, elapsed)
	}

	// Verify we made requests for all pages
	if len(mockClient.requestedPages) != 3 {
		t.Errorf("Expected requests for 3 pages, got: %v", mockClient.requestedPages)
	}
}

// Helper functions for testing timing
func start() time.Time {
	return time.Now()
}

func elapsed(startTime time.Time) time.Duration {
	return time.Since(startTime)
}
