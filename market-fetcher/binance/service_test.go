package binance

import (
	"context"
	"testing"
	"time"

	"github.com/status-im/market-proxy/config"

	"github.com/stretchr/testify/assert"
)

func TestService_SetWatchList(t *testing.T) {
	cfg := &config.Config{}
	svc := NewService(cfg)

	// Test setting watchlist
	baseSymbols := []string{"BTC", "ETH"}
	quoteSymbol := "USDT"
	svc.SetWatchList(baseSymbols, quoteSymbol)

	// Verify quotes are empty initially
	quotes := svc.GetLatestQuotes()
	assert.Equal(t, 0, len(quotes), "Quotes should be empty initially")
}

func TestService_GetLatestQuotes(t *testing.T) {
	cfg := &config.Config{}
	svc := NewService(cfg)

	// Set up watchlist
	baseSymbols := []string{"BTC", "ETH"}
	quoteSymbol := "USDT"
	svc.SetWatchList(baseSymbols, quoteSymbol)

	// Get quotes before any updates
	quotes := svc.GetLatestQuotes()
	assert.Equal(t, 0, len(quotes), "Quotes should be empty before updates")

	// Update quotes with test data
	message := []byte(`[
		{
			"e": "24hrTicker",
			"E": 1672515782136,
			"s": "BTCUSDT",
			"c": "50000.00",
			"P": "1.5",
			"v": "100.00"
		},
		{
			"e": "24hrTicker",
			"E": 1672515782136,
			"s": "ETHUSDT",
			"c": "3000.00",
			"P": "-0.5",
			"v": "1000.00"
		}
	]`)

	err := svc.quotes.UpdateQuotes(message)
	assert.NoError(t, err)

	// Get quotes
	quotes = svc.GetLatestQuotes()
	assert.Equal(t, 2, len(quotes), "Should have quotes for both symbols")

	// Verify BTC quote
	btcQuote, exists := quotes["BTC"]
	assert.True(t, exists, "BTC quote should exist")
	assert.Equal(t, 50000.00, btcQuote.Price)
	assert.Equal(t, 1.5, btcQuote.PercentChange24h)

	// Verify ETH quote
	ethQuote, exists := quotes["ETH"]
	assert.True(t, exists, "ETH quote should exist")
	assert.Equal(t, 3000.00, ethQuote.Price)
	assert.Equal(t, -0.5, ethQuote.PercentChange24h)
}

func TestService_HandleMessages(t *testing.T) {
	cfg := &config.Config{}
	svc := NewService(cfg)

	// Set up watchlist
	baseSymbols := []string{"BTC", "ETH"}
	quoteSymbol := "USDT"
	svc.SetWatchList(baseSymbols, quoteSymbol)

	// Create test message
	testMessage := []byte(`[
		{
			"e": "24hrTicker",
			"E": 1672515782136,
			"s": "BTCUSDT",
			"c": "50000.00",
			"P": "1.5",
			"v": "100.00"
		},
		{
			"e": "24hrTicker",
			"E": 1672515782136,
			"s": "ETHUSDT",
			"c": "3000.00",
			"P": "-0.5",
			"v": "1000.00"
		}
	]`)

	// Process message
	err := svc.quotes.UpdateQuotes(testMessage)
	assert.NoError(t, err)

	// Verify quotes were updated
	quotes := svc.GetLatestQuotes()
	assert.Equal(t, 2, len(quotes), "Should have quotes for both symbols")

	// Verify BTC quote
	btcQuote, exists := quotes["BTC"]
	assert.True(t, exists, "BTC quote should exist")
	assert.Equal(t, 50000.00, btcQuote.Price)
	assert.Equal(t, 1.5, btcQuote.PercentChange24h)

	// Verify ETH quote
	ethQuote, exists := quotes["ETH"]
	assert.True(t, exists, "ETH quote should exist")
	assert.Equal(t, 3000.00, ethQuote.Price)
	assert.Equal(t, -0.5, ethQuote.PercentChange24h)
}

func TestService_Reconnect(t *testing.T) {
	cfg := &config.Config{}
	svc := NewService(cfg)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start service
	err := svc.Start(ctx)
	assert.NoError(t, err)

	// Wait briefly to ensure service starts
	time.Sleep(100 * time.Millisecond)

	// Stop service
	svc.Stop()

	// Verify service is stopped
	time.Sleep(100 * time.Millisecond) // Give time for goroutines to clean up
}

func TestService_Healthy(t *testing.T) {
	cfg := &config.Config{}
	svc := NewService(cfg)

	// Initially, service should not be healthy
	assert.False(t, svc.Healthy(), "Service should not be healthy initially")

	// Setup watchlist
	baseSymbols := []string{"BTC", "ETH"}
	quoteSymbol := "USDT"
	svc.SetWatchList(baseSymbols, quoteSymbol)

	// Create test message
	testMessage := []byte(`[
		{
			"e": "24hrTicker",
			"E": 1672515782136,
			"s": "BTCUSDT",
			"c": "50000.00",
			"P": "1.5",
			"v": "100.00"
		}
	]`)

	// Process the message using the callback that was set in the constructor
	// Extract the callback from wsClient and call it directly
	err := svc.wsClient.onMessage(testMessage)
	assert.NoError(t, err)

	// Now the service should be healthy
	assert.True(t, svc.Healthy(), "Service should be healthy after processing a valid message")

	// Test with invalid message - it should not affect health status
	invalidMessage := []byte(`invalid json`)
	err = svc.wsClient.onMessage(invalidMessage)
	assert.Error(t, err)

	// Service should still be considered healthy
	assert.True(t, svc.Healthy(), "Service should remain healthy even after error")
}

func TestService_InvalidMessage(t *testing.T) {
	cfg := &config.Config{}
	svc := NewService(cfg)

	// Set up watchlist
	baseSymbols := []string{"BTC", "ETH"}
	quoteSymbol := "USDT"
	svc.SetWatchList(baseSymbols, quoteSymbol)

	// Test invalid JSON message
	invalidMessage := []byte(`invalid json`)
	err := svc.quotes.UpdateQuotes(invalidMessage)
	assert.Error(t, err, "Should return error for invalid JSON")

	// Test message with invalid number format
	invalidNumberMessage := []byte(`[
		{
			"e": "24hrTicker",
			"E": 1672515782136,
			"s": "BTCUSDT",
			"c": "invalid",
			"P": "1.5",
			"v": "100.00"
		}
	]`)
	err = svc.quotes.UpdateQuotes(invalidNumberMessage)
	assert.Error(t, err, "Should return error for invalid number format")
}
