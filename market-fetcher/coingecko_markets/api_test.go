package coingecko_markets

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
)

// MockHTTPClient is a mock implementation of the HTTP client functionality
type MockHTTPClient struct {
	// Response to return for each request
	mockResponses []*mockResponse
	// Record of requests that were executed
	executedRequests []*http.Request
	// Current response index
	currentResponse int
}

// mockResponse represents a single mocked HTTP response
type mockResponse struct {
	response *http.Response
	body     []byte
	duration time.Duration
	err      error
	matchURL string // optional URL pattern to match this response to
}

// Do implements the http.Client Do method
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.executedRequests = append(m.executedRequests, req)

	// If no mocked responses, return generic error
	if len(m.mockResponses) == 0 {
		return nil, errors.New("no mocked response available")
	}

	// If URL matching is specified, find the matching response
	if req.URL != nil {
		urlStr := req.URL.String()
		for _, resp := range m.mockResponses {
			if resp.matchURL != "" && contains(urlStr, resp.matchURL) {
				// Return matching response and its error
				return resp.response, resp.err
			}
		}
	}

	// Get next response in sequence
	resp := m.mockResponses[m.currentResponse]

	// Move to next response, cycling back to start if needed
	m.currentResponse = (m.currentResponse + 1) % len(m.mockResponses)

	return resp.response, resp.err
}

// RoundTrip implements http.RoundTripper interface
func (m *MockHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Do(req)
}

// MockTransport implements the http.RoundTripper interface for testing
type MockTransport struct {
	mockClient *MockHTTPClient
}

// RoundTrip implements the http.RoundTripper interface
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.mockClient.Do(req)
}

// MockAPIKeyManager mocks the APIKeyManagerInterface for testing
type MockAPIKeyManager struct {
	// Keys to return
	mockKeys []cg.APIKey
	// Record of keys that were marked as failed
	failedKeys []string
}

// GetAvailableKeys implements the GetAvailableKeys method for mocking
func (m *MockAPIKeyManager) GetAvailableKeys() []cg.APIKey {
	return m.mockKeys
}

// MarkKeyAsFailed implements the MarkKeyAsFailed method for mocking
func (m *MockAPIKeyManager) MarkKeyAsFailed(key string) {
	m.failedKeys = append(m.failedKeys, key)
}

// createMockResponse creates a mock HTTP response with the given status code and body
func createMockResponse(statusCode int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
		Request:    &http.Request{}, // Add non-nil request to avoid nil pointer in processResponse
	}
}

// createSampleCoinGeckoData creates sample data for testing
func createSampleCoinGeckoData() []CoinGeckoData {
	return []CoinGeckoData{
		{
			ID:                       "bitcoin",
			Symbol:                   "btc",
			Name:                     "Bitcoin",
			Image:                    "https://example.com/bitcoin.png",
			CurrentPrice:             50000.0,
			MarketCap:                1000000000.0,
			TotalVolume:              50000000.0,
			PriceChangePercentage24h: 5.0,
		},
		{
			ID:                       "ethereum",
			Symbol:                   "eth",
			Name:                     "Ethereum",
			Image:                    "https://example.com/ethereum.png",
			CurrentPrice:             3000.0,
			MarketCap:                500000000.0,
			TotalVolume:              20000000.0,
			PriceChangePercentage24h: 3.0,
		},
	}
}

// createMockHTTPClientWithRetries creates a mock HTTPClientWithRetries for testing
func createMockHTTPClientWithRetries(mockClient *MockHTTPClient) *cg.HTTPClientWithRetries {
	// Create a real http.Client that uses our mock transport
	httpClient := &http.Client{
		Transport: &MockTransport{mockClient: mockClient},
	}

	// Create a retry client with our mocked http client
	return &cg.HTTPClientWithRetries{
		Client: httpClient,
		Opts: cg.RetryOptions{
			MaxRetries:  1, // Just one attempt for tests
			BaseBackoff: 1 * time.Millisecond,
			LogPrefix:   "Test",
		},
	}
}

func TestCoinGeckoClient_FetchPage_Success(t *testing.T) {
	// Create sample data for response
	sampleData := createSampleCoinGeckoData()
	jsonData, _ := json.Marshal(sampleData)

	// Create mock HTTP client
	mockClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				response: createMockResponse(http.StatusOK, jsonData),
				body:     jsonData,
				duration: 100 * time.Millisecond,
				err:      nil,
			},
		},
	}

	// Create mock key manager with one Pro key
	mockKeyManager := &MockAPIKeyManager{
		mockKeys: []cg.APIKey{
			{Key: "test-pro-key", Type: cg.ProKey},
		},
	}

	// Create CoinGeckoClient with mocks
	client := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockClient),
	}

	// Call FetchPage
	result, err := client.FetchPage(1, 10)

	// Check for errors
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check result
	if len(result) != len(sampleData) {
		t.Errorf("Expected %d items, got %d", len(sampleData), len(result))
	}

	// Verify the first item
	if result[0].ID != "bitcoin" || result[0].Symbol != "btc" {
		t.Errorf("Expected Bitcoin data, got %v", result[0])
	}

	// Check that the HTTP client was called once
	if len(mockClient.executedRequests) != 1 {
		t.Errorf("Expected 1 HTTP request, got %d", len(mockClient.executedRequests))
	}

	// Check that no keys were marked as failed
	if len(mockKeyManager.failedKeys) != 0 {
		t.Errorf("Expected no keys to be marked as failed, got %v", mockKeyManager.failedKeys)
	}
}

func TestCoinGeckoClient_FetchPage_ErrorHandling(t *testing.T) {
	// Create mock HTTP client with error
	mockClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				response: nil,
				body:     nil,
				duration: 0,
				err:      errors.New("request failed"),
			},
		},
	}

	// Create mock key manager with one Pro key and one Demo key
	mockKeyManager := &MockAPIKeyManager{
		mockKeys: []cg.APIKey{
			{Key: "test-pro-key", Type: cg.ProKey},
			{Key: "test-demo-key", Type: cg.DemoKey},
		},
	}

	// Create CoinGeckoClient with mocks
	client := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockClient),
	}

	// Call FetchPage
	result, err := client.FetchPage(1, 10)

	// Should get an error since all keys fail
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Result should be nil
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}

	// Check that the HTTP client was called for each key
	if len(mockClient.executedRequests) != 2 {
		t.Errorf("Expected HTTP client to be called twice, got %d", len(mockClient.executedRequests))
	}

	// Check that both keys were marked as failed
	if len(mockKeyManager.failedKeys) != 2 {
		t.Errorf("Expected 2 keys to be marked as failed, got %d", len(mockKeyManager.failedKeys))
	}
}

func TestCoinGeckoClient_FetchPage_KeyFallback(t *testing.T) {
	// Create sample data for response
	sampleData := createSampleCoinGeckoData()
	jsonData, _ := json.Marshal(sampleData)

	// Create mock HTTP client with URL-based responses
	mockClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				// Response for Pro key - will fail
				response: nil,
				err:      errors.New("rate limit exceeded"),
				matchURL: "x_cg_pro_api_key",
			},
			{
				// Response for Demo key - will succeed
				response: createMockResponse(http.StatusOK, jsonData),
				body:     jsonData,
				duration: 100 * time.Millisecond,
				err:      nil,
				matchURL: "x_cg_demo_api_key",
			},
		},
	}

	// Create mock key manager with one Pro key and one Demo key
	mockKeyManager := &MockAPIKeyManager{
		mockKeys: []cg.APIKey{
			{Key: "test-pro-key", Type: cg.ProKey},
			{Key: "test-demo-key", Type: cg.DemoKey},
		},
	}

	// Create CoinGeckoClient with mocks
	client := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockClient),
	}

	// Call FetchPage
	result, err := client.FetchPage(1, 10)

	// Should not get an error since the second key succeeds
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check result
	if len(result) != len(sampleData) {
		t.Errorf("Expected %d items, got %d", len(sampleData), len(result))
	}

	// Check that the HTTP client was called twice (once for each key)
	if len(mockClient.executedRequests) != 2 {
		t.Errorf("Expected HTTP client to be called twice, got %d", len(mockClient.executedRequests))
	}

	// Check that the first key was marked as failed
	if len(mockKeyManager.failedKeys) != 1 || mockKeyManager.failedKeys[0] != "test-pro-key" {
		t.Errorf("Expected only pro key to be marked as failed, got %v", mockKeyManager.failedKeys)
	}

	// Verify the request URLs contain the correct API keys
	if len(mockClient.executedRequests) < 2 {
		t.Fatalf("Expected at least 2 executed requests, got %d", len(mockClient.executedRequests))
	}

	// First request should use Pro URL and Pro key
	firstReqURL := mockClient.executedRequests[0].URL.String()
	if !contains(firstReqURL, "x_cg_pro_api_key=test-pro-key") {
		t.Errorf("Expected first request to use Pro key, got URL: %s", firstReqURL)
	}

	// Second request should use Demo key
	secondReqURL := mockClient.executedRequests[1].URL.String()
	if !contains(secondReqURL, "x_cg_demo_api_key=test-demo-key") {
		t.Errorf("Expected second request to use Demo key, got URL: %s", secondReqURL)
	}
}

func TestCoinGeckoClient_FetchPage_InvalidJSON(t *testing.T) {
	// Create invalid JSON for response
	invalidJSON := []byte("{invalid json")

	// Create mock HTTP client
	mockClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				response: createMockResponse(http.StatusOK, invalidJSON),
				body:     invalidJSON,
				duration: 100 * time.Millisecond,
				err:      nil,
			},
		},
	}

	// Create mock key manager with one Pro key
	mockKeyManager := &MockAPIKeyManager{
		mockKeys: []cg.APIKey{
			{Key: "test-pro-key", Type: cg.ProKey},
		},
	}

	// Create CoinGeckoClient with mocks
	client := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockClient),
	}

	// Call FetchPage
	result, err := client.FetchPage(1, 10)

	// Should get a JSON parsing error
	if err == nil {
		t.Fatal("Expected JSON parsing error, got nil")
	}

	// Result should be nil
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr && len(s) > len(substr) && s != "" && bytes.Contains([]byte(s), []byte(substr))
}

// TestCoinGeckoClient_Healthy tests the Healthy method
func TestCoinGeckoClient_Healthy(t *testing.T) {
	// Create sample data for response
	sampleData := createSampleCoinGeckoData()
	jsonData, _ := json.Marshal(sampleData)

	// Create mock HTTP client
	mockClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				response: createMockResponse(http.StatusOK, jsonData),
				body:     jsonData,
				duration: 100 * time.Millisecond,
				err:      nil,
			},
		},
	}

	// Create mock key manager with one Pro key
	mockKeyManager := &MockAPIKeyManager{
		mockKeys: []cg.APIKey{
			{Key: "test-pro-key", Type: cg.ProKey},
		},
	}

	// Create CoinGeckoClient with mocks
	client := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockClient),
	}

	// Initially, the client should not be healthy
	if client.Healthy() {
		t.Fatal("Expected client to not be healthy initially")
	}

	// Call FetchPage which should update the health status
	_, err := client.FetchPage(1, 10)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// After a successful fetch, the client should be healthy
	if !client.Healthy() {
		t.Fatal("Expected client to be healthy after successful fetch")
	}

	// Create a new client with error response to test error case
	mockErrorClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				response: nil,
				body:     nil,
				duration: 0,
				err:      errors.New("request failed"),
			},
		},
	}

	errorClient := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockErrorClient),
	}

	// Initially, the client should not be healthy
	if errorClient.Healthy() {
		t.Fatal("Expected error client to not be healthy initially")
	}

	// Call FetchPage with an error, health status should remain false
	_, err = errorClient.FetchPage(1, 10)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Client should still not be healthy after a failed fetch
	if errorClient.Healthy() {
		t.Fatal("Expected client to not be healthy after failed fetch")
	}
}
