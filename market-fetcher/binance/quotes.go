package binance

import (
	"encoding/json"
	"fmt"
	"sync"
)

// QuotesManager manages quotes for watched symbols
type QuotesManager struct {
	mu sync.RWMutex
	// Map of full symbol to quote (e.g. "BTCUSDT" -> Quote)
	quotes map[string]Quote
	// Map of full symbol to base symbol (e.g. "BTCUSDT" -> "BTC")
	baseSymbols map[string]string
}

// NewQuotesManager creates a new QuotesManager
func NewQuotesManager() *QuotesManager {
	return &QuotesManager{
		quotes:      make(map[string]Quote),
		baseSymbols: make(map[string]string),
	}
}

// SetWatchList sets the list of symbols to watch with the quote symbol
func (qm *QuotesManager) SetWatchList(baseSymbols []string, quoteSymbol string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Clear existing quotes and symbols
	qm.quotes = make(map[string]Quote)
	qm.baseSymbols = make(map[string]string)

	// Set new symbols
	for _, baseSymbol := range baseSymbols {
		fullSymbol := baseSymbol + quoteSymbol
		qm.baseSymbols[fullSymbol] = baseSymbol
	}
}

// GetLatestQuotes returns the latest quotes for watched symbols
func (qm *QuotesManager) GetLatestQuotes() map[string]Quote {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	// Create a copy of the quotes map, but use base symbols as keys
	quotesCopy := make(map[string]Quote)
	for fullSymbol, quote := range qm.quotes {
		if baseSymbol, ok := qm.baseSymbols[fullSymbol]; ok {
			quotesCopy[baseSymbol] = quote
		}
	}
	return quotesCopy
}

// UpdateQuotes updates quotes from a WebSocket message
func (qm *QuotesManager) UpdateQuotes(message []byte) error {
	// Try to unmarshal as array of tickers first
	var tickers []Ticker
	if err := json.Unmarshal(message, &tickers); err != nil {
		// If array unmarshal fails, try single ticker
		var ticker Ticker
		if err := json.Unmarshal(message, &ticker); err != nil {
			return fmt.Errorf("failed to unmarshal ticker message: %v", err)
		}
		tickers = []Ticker{ticker}
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()

	for i := range tickers {
		ticker := &tickers[i]
		// Check if we're watching this symbol
		if _, ok := qm.baseSymbols[ticker.Symbol]; ok {
			// Parse values
			price, err := ticker.LastPrice.Float64()
			if err != nil {
				return fmt.Errorf("failed to parse price for %s: %v", ticker.Symbol, err)
			}

			volume24h, err := ticker.Volume24h.Float64()
			if err != nil {
				return fmt.Errorf("failed to parse volume for %s: %v", ticker.Symbol, err)
			}

			percentChange24h, err := ticker.PriceChangePercent.Float64()
			if err != nil {
				return fmt.Errorf("failed to parse price change percent for %s: %v", ticker.Symbol, err)
			}

			// Update quote using full symbol as key
			qm.quotes[ticker.Symbol] = Quote{
				Price:            price,
				Volume24h:        volume24h,
				PercentChange24h: percentChange24h,
			}
		}
	}

	return nil
}

// parseFloat64 parses a string to float64
func parseFloat64(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
