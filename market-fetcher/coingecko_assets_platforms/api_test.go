package coingecko_assets_platforms

import (
	"testing"

	"github.com/status-im/market-proxy/config"
)

func TestNewCoinGeckoClient(t *testing.T) {
	cfg := &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{"test-token"},
		},
	}

	client := NewCoinGeckoClient(cfg)

	if client == nil {
		t.Fatal("NewCoinGeckoClient returned nil")
	}

	if client.config != cfg {
		t.Error("Client config not set correctly")
	}

	if client.httpClient == nil {
		t.Error("HTTP client not initialized")
	}

	if client.keyManager == nil {
		t.Error("Key manager not initialized")
	}
}

func TestCoinGeckoClient_Healthy(t *testing.T) {
	client := NewCoinGeckoClient(&config.Config{})

	// Should be false initially
	if client.Healthy() {
		t.Error("Expected client to be unhealthy initially")
	}

	// Simulate successful fetch
	client.successfulFetch.Store(true)

	if !client.Healthy() {
		t.Error("Expected client to be healthy after successful fetch")
	}
}
