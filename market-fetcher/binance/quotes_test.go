package binance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuotesManager_SetWatchList(t *testing.T) {
	qm := NewQuotesManager()

	// Test setting watchlist
	baseSymbols := []string{"BTC", "ETH"}
	quoteSymbol := "USDT"
	qm.SetWatchList(baseSymbols, quoteSymbol)

	// Verify quotes are empty initially
	quotes := qm.GetLatestQuotes()
	assert.Equal(t, 0, len(quotes), "Quotes should be empty initially")

	// Test updating watchlist
	newSymbols := []string{"XRP", "ADA"}
	qm.SetWatchList(newSymbols, quoteSymbol)
	quotes = qm.GetLatestQuotes()
	assert.Equal(t, 0, len(quotes), "Quotes should be empty after updating watchlist")
}

func TestQuotesManager_UpdateQuotes(t *testing.T) {
	qm := NewQuotesManager()

	// Set up watchlist
	baseSymbols := []string{"BTC", "ETH"}
	quoteSymbol := "USDT"
	qm.SetWatchList(baseSymbols, quoteSymbol)

	// Test valid message
	validMessage := []byte(`[
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

	err := qm.UpdateQuotes(validMessage)
	assert.NoError(t, err)

	// Verify quotes were updated
	quotes := qm.GetLatestQuotes()
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

	// Test invalid JSON message
	invalidMessage := []byte(`invalid json`)
	err = qm.UpdateQuotes(invalidMessage)
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
	err = qm.UpdateQuotes(invalidNumberMessage)
	assert.Error(t, err, "Should return error for invalid number format")
}

func TestQuotesManager_GetLatestQuotes(t *testing.T) {
	qm := NewQuotesManager()

	// Set up watchlist
	baseSymbols := []string{"BTC", "ETH"}
	quoteSymbol := "USDT"
	qm.SetWatchList(baseSymbols, quoteSymbol)

	// Get quotes before any updates
	quotes := qm.GetLatestQuotes()
	assert.Equal(t, 0, len(quotes), "Quotes should be empty before updates")

	// Update quotes
	message := []byte(`[
		{
			"e": "24hrTicker",
			"E": 1672515782136,
			"s": "BTCUSDT",
			"c": "50000.00",
			"P": "1.5",
			"v": "100.00"
		}
	]`)
	err := qm.UpdateQuotes(message)
	assert.NoError(t, err)

	// Get quotes
	quotes = qm.GetLatestQuotes()
	assert.Equal(t, 1, len(quotes), "Should have one quote")

	// Verify quote values
	btcQuote, exists := quotes["BTC"]
	assert.True(t, exists, "BTC quote should exist")
	assert.Equal(t, 50000.00, btcQuote.Price)
	assert.Equal(t, 1.5, btcQuote.PercentChange24h)

	// Verify quotes map is a copy
	quotesCopy := qm.GetLatestQuotes()
	assert.Equal(t, quotes, quotesCopy, "Quotes maps should be equal")

	// Modify the copy and verify it doesn't affect the original
	quotes["BTC"] = Quote{Price: 60000.00}
	quotesCopy = qm.GetLatestQuotes()
	assert.Equal(t, 50000.00, quotesCopy["BTC"].Price, "Original quote should not be modified")
}
