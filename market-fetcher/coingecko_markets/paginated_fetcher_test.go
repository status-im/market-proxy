package coingecko_markets

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/status-im/market-proxy/interfaces"
)

// MockAPIClient for testing PaginatedFetcher
type MockAPIClient struct {
	// Define how many items to return per page
	itemsPerPage [][]CoinGeckoData
	// Define which pages should return errors
	errorPages map[int]error
	// Track which pages were requested
	requestedPages []int
	// Health status flag
	isHealthy bool
}

// FetchPage implements APIClient interface for mock
func (m *MockAPIClient) FetchPage(params interfaces.MarketsParams) ([][]byte, error) {
	// Record the page request
	m.requestedPages = append(m.requestedPages, params.Page)

	// Check if this page should return an error
	if err, exists := m.errorPages[params.Page]; exists {
		return nil, err
	}

	// Calculate the page index (0-based)
	pageIndex := params.Page - 1

	// Check if we have data for this page
	if pageIndex >= len(m.itemsPerPage) {
		// Return empty page if we're beyond available data
		return [][]byte{}, nil
	}

	// Get the items for this page
	pageItems := m.itemsPerPage[pageIndex]

	// Convert CoinGeckoData to [][]byte
	result := make([][]byte, 0, len(pageItems))
	for _, item := range pageItems {
		// Marshal each item to JSON bytes
		jsonBytes, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal mock data: %w", err)
		}
		result = append(result, jsonBytes)
	}

	return result, nil
}

// Healthy implements APIClient interface for mock
func (m *MockAPIClient) Healthy() bool {
	return m.isHealthy
}

// TestPaginatedFetcher_SinglePage tests fetching data that fits in a single page
func TestPaginatedFetcher_SinglePage(t *testing.T) {
	// Create mock data
	mockItems := []CoinGeckoData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
		{ID: "ripple", Symbol: "xrp", Name: "Ripple"},
	}

	// Create mock API client
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinGeckoData{mockItems},
		errorPages:   make(map[int]error),
		isHealthy:    true,
	}

	// Create fetcher with page range covering our mock data
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  10,
	}
	pageFrom := 1
	pageTo := (len(mockItems) + params.PerPage - 1) / params.PerPage // Ceiling division
	fetcher := NewPaginatedFetcher(mockClient, pageFrom, pageTo, 0, params)

	// Call FetchData
	response, err := fetcher.FetchData()

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that we got the right number of items
	if len(response) != len(mockItems) {
		t.Errorf("Expected %d items, got %d", len(mockItems), len(response))
	}

	// Check that we requested exactly one page
	if len(mockClient.requestedPages) != 1 || mockClient.requestedPages[0] != 1 {
		t.Errorf("Expected one page request for page 1, got: %v", mockClient.requestedPages)
	}

	// Check the actual items by parsing JSON
	for i, itemBytes := range response {
		var item CoinGeckoData
		if err := json.Unmarshal(itemBytes, &item); err != nil {
			t.Fatalf("Failed to unmarshal item %d: %v", i, err)
		}
		if item.ID != mockItems[i].ID {
			t.Errorf("Expected item %d to be %s, got %s", i, mockItems[i].ID, item.ID)
		}
	}
}

// TestPaginatedFetcher_MultiPage tests fetching multiple pages of data
func TestPaginatedFetcher_MultiPage(t *testing.T) {
	// Create mock data for two pages
	page1Items := []CoinGeckoData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}

	page2Items := []CoinGeckoData{
		{ID: "ripple", Symbol: "xrp", Name: "Ripple"},
		{ID: "litecoin", Symbol: "ltc", Name: "Litecoin"},
	}

	page3Items := []CoinGeckoData{
		{ID: "cardano", Symbol: "ada", Name: "Cardano"},
	}

	// Create mock API client with multiple pages of data
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinGeckoData{page1Items, page2Items, page3Items},
		errorPages:   make(map[int]error),
		isHealthy:    true,
	}

	// Create fetcher with page range requiring all pages, and minimal delay for tests
	totalItems := len(page1Items) + len(page2Items) + len(page3Items)
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  2, // Each page has limit 2
	}
	pageFrom := 1
	pageTo := (totalItems + params.PerPage - 1) / params.PerPage            // Ceiling division
	fetcher := NewPaginatedFetcher(mockClient, pageFrom, pageTo, 1, params) // 1ms delay

	// Call FetchData
	response, err := fetcher.FetchData()

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that we got the right number of items
	if len(response) != totalItems {
		t.Errorf("Expected %d items, got %d", totalItems, len(response))
	}

	// Check that we requested all three pages
	if len(mockClient.requestedPages) != 3 {
		t.Errorf("Expected three page requests, got: %v", mockClient.requestedPages)
	}

	// Check the first item (should be from page 1)
	var firstItem CoinGeckoData
	if err := json.Unmarshal(response[0], &firstItem); err != nil {
		t.Fatalf("Failed to unmarshal first item: %v", err)
	}
	if firstItem.ID != "bitcoin" {
		t.Errorf("Expected first item to be bitcoin, got %s", firstItem.ID)
	}

	// Check the last item (should be from page 3)
	var lastItem CoinGeckoData
	if err := json.Unmarshal(response[4], &lastItem); err != nil {
		t.Fatalf("Failed to unmarshal last item: %v", err)
	}
	if lastItem.ID != "cardano" {
		t.Errorf("Expected last item to be cardano, got %s", lastItem.ID)
	}
}

// TestPaginatedFetcher_Limit tests that the fetcher respects the total limit
func TestPaginatedFetcher_Limit(t *testing.T) {
	// Create mock data with many items
	page1Items := []CoinGeckoData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
		{ID: "ripple", Symbol: "xrp", Name: "Ripple"},
	}

	page2Items := []CoinGeckoData{
		{ID: "litecoin", Symbol: "ltc", Name: "Litecoin"},
		{ID: "cardano", Symbol: "ada", Name: "Cardano"},
		{ID: "polkadot", Symbol: "dot", Name: "Polkadot"},
	}

	// Create mock API client with two pages of data
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinGeckoData{page1Items, page2Items},
		errorPages:   make(map[int]error),
		isHealthy:    true,
	}

	// Create fetcher with a page range that fetches only the first page (3 items out of 6 available)
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  3, // Each page has limit 3
	}
	pageFrom := 1
	pageTo := 1                                                             // Only fetch first page
	fetcher := NewPaginatedFetcher(mockClient, pageFrom, pageTo, 0, params) // no delay

	// Call FetchData
	response, err := fetcher.FetchData()

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that we got items from only the first page (3 items)
	expectedItems := 3 // Items from first page only
	if len(response) != expectedItems {
		t.Errorf("Expected %d items from first page only, got %d", expectedItems, len(response))
	}

	// Check that we requested only one page
	if len(mockClient.requestedPages) != 1 {
		t.Errorf("Expected one page request, got: %v", mockClient.requestedPages)
	}

	// Check the first and last items to ensure they're correct
	var firstItem CoinGeckoData
	if err := json.Unmarshal(response[0], &firstItem); err != nil {
		t.Fatalf("Failed to unmarshal first item: %v", err)
	}
	if firstItem.ID != "bitcoin" {
		t.Errorf("Expected first item to be bitcoin, got %s", firstItem.ID)
	}

	var lastItem CoinGeckoData
	if err := json.Unmarshal(response[2], &lastItem); err != nil {
		t.Fatalf("Failed to unmarshal last item: %v", err)
	}
	if lastItem.ID != "ripple" {
		t.Errorf("Expected last item to be ripple, got %s", lastItem.ID)
	}
}

// TestPaginatedFetcher_ErrorFirstPage tests handling errors on the first page
func TestPaginatedFetcher_ErrorFirstPage(t *testing.T) {
	// Create mock API client with error on first page
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinGeckoData{},
		errorPages:   map[int]error{1: errors.New("API error on first page")},
		isHealthy:    true,
	}

	// Create fetcher
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  5,
	}
	pageFrom := 1
	pageTo := (10 + params.PerPage - 1) / params.PerPage // Ceiling division
	fetcher := NewPaginatedFetcher(mockClient, pageFrom, pageTo, 0, params)

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
	page1Items := []CoinGeckoData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}

	// Create mock API client with error on second page
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinGeckoData{page1Items},
		errorPages:   map[int]error{2: errors.New("API error on second page")},
		isHealthy:    true,
	}

	// Create fetcher with page range requiring multiple pages
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  2,
	}
	pageFrom := 1
	pageTo := (4 + params.PerPage - 1) / params.PerPage                     // Ceiling division
	fetcher := NewPaginatedFetcher(mockClient, pageFrom, pageTo, 0, params) // need 2 pages

	// Call FetchData - should return partial data
	response, err := fetcher.FetchData()

	// Should not get an error - should return partial data
	if err != nil {
		t.Fatalf("Expected no error (partial data), got: %v", err)
	}

	// Should have the data from the first page
	if len(response) != len(page1Items) {
		t.Errorf("Expected %d items from first page, got %d", len(page1Items), len(response))
	}

	// Should have tried both pages
	if len(mockClient.requestedPages) != 2 {
		t.Errorf("Expected two page requests, got: %v", mockClient.requestedPages)
	}
}

// TestPaginatedFetcher_ZeroLimit tests that a zero limit request handles appropriately
func TestPaginatedFetcher_ZeroLimit(t *testing.T) {
	// Create mock API client
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinGeckoData{},
		errorPages:   make(map[int]error),
		isHealthy:    true,
	}

	// Create fetcher with zero pages (pageFrom > pageTo means no pages)
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  10,
	}
	pageFrom := 1
	pageTo := 0 // No pages to fetch
	fetcher := NewPaginatedFetcher(mockClient, pageFrom, pageTo, 0, params)

	// Call FetchData
	response, err := fetcher.FetchData()

	// Should not get an error
	if err != nil {
		t.Errorf("Expected no error with zero limit, got: %v", err)
	}

	// Should get empty response
	if len(response) != 0 {
		t.Errorf("Expected 0 items with zero limit, got %d", len(response))
	}

	// Should not make any API requests
	if len(mockClient.requestedPages) != 0 {
		t.Errorf("Expected no page requests with zero limit, got: %v", mockClient.requestedPages)
	}
}

// TestPaginatedFetcher_LargeRequest tests fetching more data than actually available
func TestPaginatedFetcher_LargeRequest(t *testing.T) {
	// Create mock data with one page
	mockItems := []CoinGeckoData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}

	// Create mock API client with only one page of data
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinGeckoData{mockItems},
		errorPages:   make(map[int]error),
		isHealthy:    true,
	}

	// Create fetcher with a large page range
	limit := 100 // Much more than available
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  10,
	}
	pageFrom := 1
	pageTo := (limit + params.PerPage - 1) / params.PerPage // Ceiling division
	fetcher := NewPaginatedFetcher(mockClient, pageFrom, pageTo, 0, params)

	// Call FetchData
	response, err := fetcher.FetchData()

	// Should not get an error
	if err != nil {
		t.Fatalf("Expected no error with large request, got: %v", err)
	}

	// Should get only the available items
	if len(response) != len(mockItems) {
		t.Errorf("Expected %d available items, got %d", len(mockItems), len(response))
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
	page1Items := []CoinGeckoData{{ID: "bitcoin"}}
	page2Items := []CoinGeckoData{{ID: "ethereum"}}
	page3Items := []CoinGeckoData{{ID: "ripple"}}

	// Create mock API client with multiple pages
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinGeckoData{page1Items, page2Items, page3Items},
		errorPages:   make(map[int]error),
		isHealthy:    true,
	}

	// Create fetcher with a significant delay (100ms for test)
	delay := 100 // 100ms delay between pages
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  1,
	}
	pageFrom := 1
	pageTo := 3
	fetcher := NewPaginatedFetcher(mockClient, pageFrom, pageTo, delay, params)

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
	page1Items := []CoinGeckoData{{ID: "bitcoin"}}
	page2Items := []CoinGeckoData{{ID: "ethereum"}}
	page3Items := []CoinGeckoData{{ID: "ripple"}}

	// Create mock API client with multiple pages
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinGeckoData{page1Items, page2Items, page3Items},
		errorPages:   make(map[int]error),
		isHealthy:    true,
	}

	// Create fetcher with zero delay
	params := interfaces.MarketsParams{
		Currency: "usd",
		Order:    "market_cap_desc",
		PerPage:  1,
	}
	pageFrom := 1
	pageTo := 3
	fetcher := NewPaginatedFetcher(mockClient, pageFrom, pageTo, 0, params)

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
