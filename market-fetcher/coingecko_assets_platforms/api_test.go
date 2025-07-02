package coingecko_assets_platforms

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
)

// MockHTTPClient is a mock implementation of the HTTP client functionality
type MockHTTPClient struct {
	mockResponses    []*mockResponse
	executedRequests []*http.Request
	currentResponse  int
}

type mockResponse struct {
	response *http.Response
	body     []byte
	duration time.Duration
	err      error
	matchURL string
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.executedRequests = append(m.executedRequests, req)

	if len(m.mockResponses) == 0 {
		return nil, errors.New("no mocked response available")
	}

	if req.URL != nil {
		urlStr := req.URL.String()
		for _, resp := range m.mockResponses {
			if resp.matchURL != "" && contains(urlStr, resp.matchURL) {
				return resp.response, resp.err
			}
		}
	}

	resp := m.mockResponses[m.currentResponse]
	m.currentResponse = (m.currentResponse + 1) % len(m.mockResponses)

	return resp.response, resp.err
}

type MockTransport struct {
	mockClient *MockHTTPClient
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.mockClient.Do(req)
}

// MockAPIKeyManager mocks the APIKeyManagerInterface for testing
type MockAPIKeyManager struct {
	mockKeys   []cg.APIKey
	failedKeys []string
}

func (m *MockAPIKeyManager) GetAvailableKeys() []cg.APIKey {
	return m.mockKeys
}

func (m *MockAPIKeyManager) MarkKeyAsFailed(key string) {
	m.failedKeys = append(m.failedKeys, key)
}

func createMockResponse(statusCode int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
		Request:    &http.Request{},
	}
}

func createMockHTTPClientWithRetries(mockClient *MockHTTPClient) *cg.HTTPClientWithRetries {
	httpClient := &http.Client{
		Transport: &MockTransport{mockClient: mockClient},
	}

	return &cg.HTTPClientWithRetries{
		Client: httpClient,
		Opts: cg.RetryOptions{
			MaxRetries:  1,
			BaseBackoff: 1 * time.Millisecond,
			LogPrefix:   "Test",
		},
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

func TestCoinGeckoClient_FetchAssetsPlatforms_Success(t *testing.T) {
	sampleData := []byte(`[{"id":"ethereum","chain_identifier":1,"name":"Ethereum"},{"id":"polygon-pos","chain_identifier":137,"name":"Polygon POS"}]`)

	mockClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				response: createMockResponse(http.StatusOK, sampleData),
				body:     sampleData,
				duration: 100 * time.Millisecond,
				err:      nil,
			},
		},
	}

	mockKeyManager := &MockAPIKeyManager{
		mockKeys: []cg.APIKey{
			{Key: "test-pro-key", Type: cg.ProKey},
		},
	}

	client := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockClient),
	}

	params := AssetsPlatformsParams{}
	result, err := client.FetchAssetsPlatforms(params)

	assert.NoError(t, err)
	assert.Equal(t, sampleData, result)
	assert.True(t, client.Healthy())
}

func TestCoinGeckoClient_FetchAssetsPlatforms_WithFilter(t *testing.T) {
	sampleData := []byte(`[{"id":"ethereum","chain_identifier":1,"name":"Ethereum"}]`)

	mockClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				response: createMockResponse(http.StatusOK, sampleData),
				body:     sampleData,
				duration: 100 * time.Millisecond,
				err:      nil,
			},
		},
	}

	mockKeyManager := &MockAPIKeyManager{
		mockKeys: []cg.APIKey{
			{Key: "test-key", Type: cg.ProKey},
		},
	}

	client := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockClient),
	}

	params := AssetsPlatformsParams{Filter: "ethereum"}
	result, err := client.FetchAssetsPlatforms(params)

	assert.NoError(t, err)
	assert.Equal(t, sampleData, result)

	// Check that the filter parameter was added to the URL
	assert.Len(t, mockClient.executedRequests, 1)
	assert.Contains(t, mockClient.executedRequests[0].URL.RawQuery, "filter=ethereum")
}

func TestCoinGeckoClient_FetchAssetsPlatforms_ErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		mockResponse *mockResponse
		expectedErr  string
	}{
		{
			name: "HTTP error",
			mockResponse: &mockResponse{
				response: nil,
				body:     nil,
				err:      errors.New("network error"),
			},
			expectedErr: "all API keys failed",
		},
		{
			name: "Bad status code",
			mockResponse: &mockResponse{
				response: createMockResponse(http.StatusInternalServerError, []byte("server error")),
				body:     []byte("server error"),
				err:      nil,
			},
			expectedErr: "all API keys failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				mockResponses: []*mockResponse{tt.mockResponse},
			}

			mockKeyManager := &MockAPIKeyManager{
				mockKeys: []cg.APIKey{
					{Key: "test-key", Type: cg.ProKey},
				},
			}

			client := &CoinGeckoClient{
				config:     &config.Config{},
				keyManager: mockKeyManager,
				httpClient: createMockHTTPClientWithRetries(mockClient),
			}

			params := AssetsPlatformsParams{}
			result, err := client.FetchAssetsPlatforms(params)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Nil(t, result)
		})
	}
}

func TestCoinGeckoClient_FetchAssetsPlatforms_KeyFallback(t *testing.T) {
	sampleData := []byte(`[{"id":"ethereum","name":"Ethereum"}]`)

	// First key fails, second succeeds
	mockClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				response: nil,
				err:      errors.New("rate limited"),
			},
			{
				response: createMockResponse(http.StatusOK, sampleData),
				body:     sampleData,
				err:      nil,
			},
		},
	}

	mockKeyManager := &MockAPIKeyManager{
		mockKeys: []cg.APIKey{
			{Key: "failing-key", Type: cg.ProKey},
			{Key: "working-key", Type: cg.DemoKey},
		},
	}

	client := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockClient),
	}

	params := AssetsPlatformsParams{}
	result, err := client.FetchAssetsPlatforms(params)

	assert.NoError(t, err)
	assert.Equal(t, sampleData, result)

	// Check that the failed key was marked as such
	assert.Contains(t, mockKeyManager.failedKeys, "failing-key")
}

func TestCoinGeckoClient_FetchAssetsPlatforms_NoKey(t *testing.T) {
	sampleData := []byte(`[{"id":"ethereum","name":"Ethereum"}]`)

	mockClient := &MockHTTPClient{
		mockResponses: []*mockResponse{
			{
				response: createMockResponse(http.StatusOK, sampleData),
				body:     sampleData,
				err:      nil,
			},
		},
	}

	mockKeyManager := &MockAPIKeyManager{
		mockKeys: []cg.APIKey{
			{Key: "", Type: cg.NoKey},
		},
	}

	client := &CoinGeckoClient{
		config:     &config.Config{},
		keyManager: mockKeyManager,
		httpClient: createMockHTTPClientWithRetries(mockClient),
	}

	params := AssetsPlatformsParams{}
	result, err := client.FetchAssetsPlatforms(params)

	assert.NoError(t, err)
	assert.Equal(t, sampleData, result)

	// Check that no API key was added to the request
	assert.Len(t, mockClient.executedRequests, 1)
	url := mockClient.executedRequests[0].URL
	assert.NotContains(t, url.RawQuery, "x_cg_pro_api_key")
	assert.NotContains(t, url.RawQuery, "x_cg_demo_api_key")
}

func TestCoinGeckoClient_Healthy(t *testing.T) {
	tests := []struct {
		name            string
		successfulFetch bool
		expected        bool
	}{
		{
			name:            "Healthy after successful fetch",
			successfulFetch: true,
			expected:        true,
		},
		{
			name:            "Unhealthy before any fetch",
			successfulFetch: false,
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &CoinGeckoClient{}

			if tt.successfulFetch {
				client.successfulFetch.Store(true)
			}

			result := client.Healthy()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewCoinGeckoClient(t *testing.T) {
	config := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{"test-token"},
		},
	}

	client := NewCoinGeckoClient(config)

	assert.NotNil(t, client)
	assert.Equal(t, config, client.config)
	assert.NotNil(t, client.keyManager)
	assert.NotNil(t, client.httpClient)
	assert.False(t, client.Healthy()) // Should be false initially
}
