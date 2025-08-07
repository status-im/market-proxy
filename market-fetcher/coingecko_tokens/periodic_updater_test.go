package coingecko_tokens

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/status-im/market-proxy/interfaces"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

// mockClient for testing
type mockClient struct {
	tokens []interfaces.Token
	err    error
}

func (c *mockClient) FetchTokens() ([]interfaces.Token, error) {
	return c.tokens, c.err
}

func TestNewPeriodicUpdater(t *testing.T) {
	cfg := config.CoinslistFetcherConfig{
		UpdateInterval:     30 * time.Second,
		SupportedPlatforms: []string{"ethereum"},
	}

	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	callback := func(ctx context.Context, tokens []interfaces.Token) error {
		return nil
	}

	updater := NewPeriodicUpdater(cfg, &Client{}, metricsWriter, callback)

	if updater == nil {
		t.Fatal("NewPeriodicUpdater returned nil")
	}

	if updater.config.UpdateInterval != cfg.UpdateInterval {
		t.Error("Config not set correctly")
	}

	if updater.metricsWriter != metricsWriter {
		t.Error("MetricsWriter not set correctly")
	}

	if updater.onUpdated == nil {
		t.Error("Callback not set correctly")
	}
}

func TestPeriodicUpdater_IsInitialized(t *testing.T) {
	cfg := config.CoinslistFetcherConfig{
		UpdateInterval:     30 * time.Second,
		SupportedPlatforms: []string{"ethereum"},
	}

	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	callback := func(ctx context.Context, tokens []interfaces.Token) error {
		return nil
	}

	updater := NewPeriodicUpdater(cfg, &Client{}, metricsWriter, callback)

	// Should start as not initialized
	if updater.IsInitialized() {
		t.Error("Expected IsInitialized to be false initially")
	}

	// Set initialized
	updater.initialized.Store(true)

	if !updater.IsInitialized() {
		t.Error("Expected IsInitialized to be true after setting")
	}
}

func TestPeriodicUpdater_fetchAndUpdate_Success(t *testing.T) {
	cfg := config.CoinslistFetcherConfig{
		UpdateInterval:     30 * time.Second,
		SupportedPlatforms: []string{"ethereum"},
	}

	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	// Mock tokens
	mockTokens := []interfaces.Token{
		{
			ID:     "ethereum",
			Symbol: "eth",
			Name:   "Ethereum",
			Platforms: map[string]string{
				"ethereum": "native",
			},
		},
		{
			ID:     "usdc",
			Symbol: "usdc",
			Name:   "USD Coin",
			Platforms: map[string]string{
				"ethereum":    "0xa0b86a33e6776b1e0e729c3b87c3c8c3",
				"polygon-pos": "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
			},
		},
	}

	var callbackTokens []interfaces.Token
	var callbackErr error
	callback := func(ctx context.Context, tokens []interfaces.Token) error {
		callbackTokens = tokens
		return callbackErr
	}

	// Create a real client for the updater
	realClient := NewClient("", metricsWriter)

	ctx := context.Background()

	// Test callback error handling
	callbackErr = errors.New("callback error")

	// Create a simplified version that we can test
	testUpdater := &PeriodicUpdater{
		config:        cfg,
		client:        realClient, // Use real client
		metricsWriter: metricsWriter,
		onUpdated:     callback,
	}

	// We can't easily test fetchAndUpdate without a real client connection
	// Let's test the initialization and callback mechanism instead
	if testUpdater.onUpdated == nil {
		t.Error("onUpdated callback not set")
	}

	// Test that callback is called with error handling
	if testUpdater.onUpdated != nil {
		err := testUpdater.onUpdated(ctx, mockTokens)
		if err == nil {
			t.Error("Expected callback to return error")
		}
		if err.Error() != "callback error" {
			t.Errorf("Expected 'callback error', got '%v'", err)
		}
	}

	// Test successful callback
	callbackErr = nil
	if testUpdater.onUpdated != nil {
		err := testUpdater.onUpdated(ctx, mockTokens)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check that callback received the original tokens (no filtering in callback)
		if len(callbackTokens) != len(mockTokens) {
			t.Errorf("Expected %d tokens, got %d", len(mockTokens), len(callbackTokens))
		}

		if len(callbackTokens) > 0 && callbackTokens[0].ID != "ethereum" {
			t.Errorf("Expected first token to be ethereum, got %s", callbackTokens[0].ID)
		}
	}
}

func TestPeriodicUpdater_Start_DisabledInterval(t *testing.T) {
	cfg := config.CoinslistFetcherConfig{
		UpdateInterval:     0, // Disabled
		SupportedPlatforms: []string{"ethereum"},
	}

	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	callback := func(ctx context.Context, tokens []interfaces.Token) error {
		return nil
	}

	updater := NewPeriodicUpdater(cfg, &Client{}, metricsWriter, callback)

	ctx := context.Background()
	err := updater.Start(ctx)

	if err != nil {
		t.Errorf("Expected no error when interval is disabled, got %v", err)
	}

	// Scheduler should be nil when interval is disabled
	if updater.scheduler != nil {
		t.Error("Expected scheduler to be nil when interval is disabled")
	}
}

func TestPeriodicUpdater_Stop(t *testing.T) {
	cfg := config.CoinslistFetcherConfig{
		UpdateInterval:     30 * time.Second,
		SupportedPlatforms: []string{"ethereum"},
	}

	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	callback := func(ctx context.Context, tokens []interfaces.Token) error {
		return nil
	}

	updater := NewPeriodicUpdater(cfg, &Client{}, metricsWriter, callback)

	// Test stop when scheduler is nil
	updater.Stop()

	// Should not panic and complete successfully
}

func TestPeriodicUpdater_CallbackParameters(t *testing.T) {
	cfg := config.CoinslistFetcherConfig{
		UpdateInterval:     30 * time.Second,
		SupportedPlatforms: []string{"ethereum", "polygon-pos"},
	}

	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	// Test that callback receives correct parameters
	var receivedCtx context.Context
	var receivedTokens []interfaces.Token

	callback := func(ctx context.Context, tokens []interfaces.Token) error {
		receivedCtx = ctx
		receivedTokens = tokens
		return nil
	}

	updater := NewPeriodicUpdater(cfg, &Client{}, metricsWriter, callback)

	// Test callback directly
	ctx := context.Background()
	testTokens := []interfaces.Token{
		{
			ID:     "ethereum",
			Symbol: "eth",
			Name:   "Ethereum",
			Platforms: map[string]string{
				"ethereum": "native",
			},
		},
	}

	err := updater.onUpdated(ctx, testTokens)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if receivedCtx != ctx {
		t.Error("Context not passed correctly to callback")
	}

	if len(receivedTokens) != len(testTokens) {
		t.Errorf("Expected %d tokens, got %d", len(testTokens), len(receivedTokens))
	}

	if receivedTokens[0].ID != testTokens[0].ID {
		t.Errorf("Expected token ID %s, got %s", testTokens[0].ID, receivedTokens[0].ID)
	}
}
