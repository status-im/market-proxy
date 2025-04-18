package coingecko

import (
	"errors"
	"testing"
)

// MockAPIClient is a mock implementation of the APIClient interface for testing
type MockAPIClient struct {
	// Define how many items to return per page
	itemsPerPage [][]CoinData
	// Define which pages should return errors
	errorPages map[int]error
	// Track which pages were requested
	requestedPages []int
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
	fetcher := NewPaginatedFetcher(mockClient, len(mockItems), 10)

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

	// Create fetcher with total limit requiring all pages
	totalItems := len(page1Items) + len(page2Items) + len(page3Items)
	fetcher := NewPaginatedFetcher(mockClient, totalItems, 2) // Each page has limit 2

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
	// Create mock data for two pages
	page1Items := []CoinData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}

	page2Items := []CoinData{
		{ID: "ripple", Symbol: "xrp", Name: "Ripple"},
		{ID: "litecoin", Symbol: "ltc", Name: "Litecoin"},
	}

	// Create mock API client
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{page1Items, page2Items},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with total limit less than all available data
	totalLimit := 3                                           // We want 3 out of 4 available items
	fetcher := NewPaginatedFetcher(mockClient, totalLimit, 2) // Each page has limit 2

	// Call FetchData
	response, err := fetcher.FetchData()

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that we got the right number of items (respecting the limit)
	if len(response.Data) != totalLimit {
		t.Errorf("Expected %d items (due to limit), got %d", totalLimit, len(response.Data))
	}

	// Check that we requested both pages
	if len(mockClient.requestedPages) != 2 {
		t.Errorf("Expected two page requests, got: %v", mockClient.requestedPages)
	}

	// Check the items - should include all from page 1 and one from page 2
	if response.Data[0].ID != "bitcoin" ||
		response.Data[1].ID != "ethereum" ||
		response.Data[2].ID != "ripple" {
		t.Errorf("Unexpected items in result: %v", response.Data)
	}
}

// TestPaginatedFetcher_EmptyPage tests handling of empty pages
func TestPaginatedFetcher_EmptyPage(t *testing.T) {
	// Create mock data with a non-empty page followed by an empty page
	page1Items := []CoinData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}

	emptyPage := []CoinData{} // Empty page

	// Create mock API client
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{page1Items, emptyPage},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with large total limit
	fetcher := NewPaginatedFetcher(mockClient, 10, 2) // Each page has limit 2

	// Call FetchData
	response, err := fetcher.FetchData()

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that we got all the items from the non-empty pages
	if len(response.Data) != len(page1Items) {
		t.Errorf("Expected %d items, got %d", len(page1Items), len(response.Data))
	}

	// Check that we tried both pages
	if len(mockClient.requestedPages) != 2 {
		t.Errorf("Expected two page requests, got: %v", mockClient.requestedPages)
	}
}

// TestPaginatedFetcher_ErrorFirstPage tests handling of errors on the first page
func TestPaginatedFetcher_ErrorFirstPage(t *testing.T) {
	// Create mock API client with error on first page
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{},
		errorPages:   map[int]error{1: errors.New("API error on first page")},
	}

	// Create fetcher
	fetcher := NewPaginatedFetcher(mockClient, 5, 2)

	// Call FetchData
	response, err := fetcher.FetchData()

	// Should get an error
	if err == nil {
		t.Error("Expected error for first page failure, got nil")
	}

	// Response should be nil
	if response != nil {
		t.Errorf("Expected nil response, got: %v", response)
	}

	// Check that we requested only the first page
	if len(mockClient.requestedPages) != 1 || mockClient.requestedPages[0] != 1 {
		t.Errorf("Expected one page request for page 1, got: %v", mockClient.requestedPages)
	}
}

// TestPaginatedFetcher_ErrorLaterPage tests handling of errors after some pages have been fetched
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
	fetcher := NewPaginatedFetcher(mockClient, 5, 2)

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
	fetcher := NewPaginatedFetcher(mockClient, 0, 10)

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
	// Create mock data for only one page
	page1Items := []CoinData{
		{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin"},
		{ID: "ethereum", Symbol: "eth", Name: "Ethereum"},
	}

	// Create mock API client that returns empty result for page 2 and beyond
	mockClient := &MockAPIClient{
		itemsPerPage: [][]CoinData{page1Items},
		errorPages:   make(map[int]error),
	}

	// Create fetcher with total limit much larger than available data
	fetcher := NewPaginatedFetcher(mockClient, 50, 2) // We want 50 items but only 2 are available

	// Call FetchData
	response, err := fetcher.FetchData()

	// Should not get an error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should get all available data even if less than requested
	if len(response.Data) != len(page1Items) {
		t.Errorf("Expected %d available items, got %d", len(page1Items), len(response.Data))
	}

	// Should stop requesting pages once an empty page is received
	// (it might request more than 2 pages depending on implementation,
	// but it should eventually stop)
	if len(mockClient.requestedPages) <= 1 {
		t.Errorf("Expected multiple page requests until empty page, got: %v", mockClient.requestedPages)
	}
}
