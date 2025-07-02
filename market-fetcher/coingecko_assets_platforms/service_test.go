package coingecko_assets_platforms

import (
	"context"
	"fmt"
	"testing"

	"github.com/status-im/market-proxy/config"
)

type mockAPIClient struct {
	shouldFail      bool
	shouldBeHealthy bool
	response        AssetsPlatformsResponse
}

func (m *mockAPIClient) FetchAssetsPlatforms(params AssetsPlatformsParams) (AssetsPlatformsResponse, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock error")
	}
	return m.response, nil
}

func (m *mockAPIClient) Healthy() bool {
	return m.shouldBeHealthy
}

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

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.config != config {
		t.Error("Service config not set correctly")
	}

	if service.client == nil {
		t.Error("Client should be initialized")
	}
}

func TestService_StartStop(t *testing.T) {
	config := createTestConfig()
	service := NewService(config)

	// Test starting
	err := service.Start(context.Background())
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Test stopping
	service.Stop()
	// Stop should not panic or fail
}

func TestService_AssetsPlatforms(t *testing.T) {
	config := createTestConfig()
	service := NewService(config)

	// Mock successful response
	mockData := []interface{}{
		map[string]interface{}{"id": "ethereum", "name": "Ethereum"},
		map[string]interface{}{"id": "polygon-pos", "name": "Polygon"},
	}

	mockClient := &mockAPIClient{
		shouldFail:      false,
		shouldBeHealthy: true,
		response:        mockData,
	}
	service.client = mockClient

	// Test successful call
	result, err := service.AssetsPlatforms(AssetsPlatformsParams{Filter: "nft"})
	if err != nil {
		t.Fatalf("AssetsPlatforms failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}

	// Test with API error
	mockClient.shouldFail = true
	_, err = service.AssetsPlatforms(AssetsPlatformsParams{})
	if err == nil {
		t.Error("AssetsPlatforms should fail when API fails")
	}
}

func TestService_Healthy(t *testing.T) {
	config := createTestConfig()
	service := NewService(config)

	// Mock healthy client
	mockClient := &mockAPIClient{shouldBeHealthy: true}
	service.client = mockClient

	if !service.Healthy() {
		t.Error("Service should be healthy when client is healthy")
	}

	// Mock unhealthy client
	mockClient.shouldBeHealthy = false
	if service.Healthy() {
		t.Error("Service should not be healthy when client is unhealthy")
	}

	// Test with nil client
	service.client = nil
	if service.Healthy() {
		t.Error("Service should not be healthy when client is nil")
	}
}
