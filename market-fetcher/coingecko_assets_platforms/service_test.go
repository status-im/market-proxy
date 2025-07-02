package coingecko_assets_platforms

import (
	"context"
	"errors"
	"testing"

	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPIClient implements APIClient interface for testing
type MockAPIClient struct {
	mock.Mock
}

func (m *MockAPIClient) FetchAssetsPlatforms(params AssetsPlatformsParams) ([]byte, error) {
	args := m.Called(params)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockAPIClient) Healthy() bool {
	args := m.Called()
	return args.Bool(0)
}

// Test data constants
var (
	samplePlatformsData = []byte(`[{"id":"ethereum","chain_identifier":1,"name":"Ethereum","shortname":"Ethereum"},{"id":"polygon-pos","chain_identifier":137,"name":"Polygon POS","shortname":"Polygon"}]`)
	emptyPlatformsData  = []byte(`[]`)
)

func createTestConfig() *config.Config {
	return &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{"test-token"},
		},
	}
}

func TestNewService(t *testing.T) {
	config := createTestConfig()
	service := NewService(config)

	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
	assert.NotNil(t, service.client)
	assert.NotNil(t, service.metricsWriter)
}

func TestService_Start(t *testing.T) {
	service := NewService(createTestConfig())
	err := service.Start(context.Background())
	assert.NoError(t, err)
}

func TestService_Stop(t *testing.T) {
	service := NewService(createTestConfig())
	assert.NotPanics(t, func() {
		service.Stop()
	})
}

func TestService_AssetsPlatforms(t *testing.T) {
	tests := []struct {
		name          string
		params        AssetsPlatformsParams
		mockData      []byte
		mockError     error
		expectedData  []byte
		expectedError string
	}{
		{
			name:          "Success with filter",
			params:        AssetsPlatformsParams{Filter: "ethereum"},
			mockData:      samplePlatformsData,
			mockError:     nil,
			expectedData:  samplePlatformsData,
			expectedError: "",
		},
		{
			name:          "Success without filter",
			params:        AssetsPlatformsParams{},
			mockData:      samplePlatformsData,
			mockError:     nil,
			expectedData:  samplePlatformsData,
			expectedError: "",
		},
		{
			name:          "Empty data",
			params:        AssetsPlatformsParams{},
			mockData:      emptyPlatformsData,
			mockError:     nil,
			expectedData:  emptyPlatformsData,
			expectedError: "",
		},
		{
			name:          "API error",
			params:        AssetsPlatformsParams{},
			mockData:      nil,
			mockError:     errors.New("API error"),
			expectedData:  nil,
			expectedError: "failed to fetch assets platforms: API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAPIClient{}
			mockClient.On("FetchAssetsPlatforms", tt.params).Return(tt.mockData, tt.mockError)

			service := &Service{
				config: createTestConfig(),
				client: mockClient,
			}

			result, err := service.AssetsPlatforms(tt.params)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedData, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestService_Healthy(t *testing.T) {
	tests := []struct {
		name           string
		client         APIClient
		expectedHealth bool
	}{
		{
			name: "Healthy client",
			client: func() APIClient {
				mockClient := &MockAPIClient{}
				mockClient.On("Healthy").Return(true)
				return mockClient
			}(),
			expectedHealth: true,
		},
		{
			name: "Unhealthy client",
			client: func() APIClient {
				mockClient := &MockAPIClient{}
				mockClient.On("Healthy").Return(false)
				return mockClient
			}(),
			expectedHealth: false,
		},
		{
			name:           "Nil client",
			client:         nil,
			expectedHealth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &Service{
				config: createTestConfig(),
				client: tt.client,
			}

			result := service.Healthy()
			assert.Equal(t, tt.expectedHealth, result)

			if mockClient, ok := tt.client.(*MockAPIClient); ok {
				mockClient.AssertExpectations(t)
			}
		})
	}
}
