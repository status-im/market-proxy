package binance

import (
	"encoding/json"
)

// WebSocketMessage represents a generic WebSocket message
type WebSocketMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// TickerMessage represents a Binance WebSocket ticker message
type TickerMessage struct {
	Symbol string `json:"s"`
	Price  string `json:"p"`
}

// PriceUpdate represents a processed price update
type PriceUpdate struct {
	Symbol string
	Price  float64
}

// WebSocketService interface defines the methods for Binance WebSocket service
type WebSocketService interface {
	Connect() error
	Disconnect() error
	SubscribeToSymbols(symbols []string) error
	GetPriceUpdates() <-chan PriceUpdate
}

// Quote represents price data for a symbol
type Quote struct {
	Price            float64 `json:"price"`
	PercentChange24h float64 `json:"percent_change_24h"`
}

// Ticker represents a Binance WebSocket ticker message
type Ticker struct {
	EventType          string      `json:"e"` // Event type
	EventTime          int64       `json:"E"` // Event time
	Symbol             string      `json:"s"` // Symbol
	PriceChange        json.Number `json:"p"` // Price change
	PriceChangePercent json.Number `json:"P"` // Price change percent
	LastPrice          json.Number `json:"c"` // Last price
	Volume24h          json.Number `json:"v"` // Total traded base asset volume
	OpenPrice          json.Number `json:"o"` // Open price
	HighPrice          json.Number `json:"h"` // High price
	LowPrice           json.Number `json:"l"` // Low price
	QuoteVolume        json.Number `json:"q"` // Quote asset volume
	OpenTime           int64       `json:"O"` // Open time
	CloseTime          int64       `json:"C"` // Close time
	FirstTradeID       int64       `json:"F"` // First trade ID
	LastTradeID        int64       `json:"L"` // Last trade ID
	TradeCount         int64       `json:"n"` // Number of trades
}
