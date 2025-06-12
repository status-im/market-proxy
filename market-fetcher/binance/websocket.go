package binance

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	BASE_WS_URL = "wss://data-stream.binance.vision/ws/!ticker@arr"
	// Connection timeouts
	PONG_TIMEOUT = 60 * time.Second
)

// WebSocketClient manages WebSocket connection to Binance
type WebSocketClient struct {
	onMessage func(message []byte) error
	wsURL     string
	mu        sync.Mutex
	started   atomic.Bool

	// The underlying simple WebSocket client
	client *SimpleWebSocketClient
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(onMessage func(message []byte) error, wsURL string) *WebSocketClient {
	if wsURL == "" {
		wsURL = BASE_WS_URL
	}
	return &WebSocketClient{
		onMessage: onMessage,
		wsURL:     wsURL,
	}
}

// Connect establishes connection to Binance WebSocket API with the given context
// It's safe to call this method multiple times - it will only connect once unless Close() was called
func (wsc *WebSocketClient) Connect(ctx context.Context) error {
	if wsc.started.Load() {
		return nil
	}
	wsc.started.Store(true)

	// Start a new connection
	return wsc.reconnect(ctx)
}

// connect establishes a new connection (must be called with mutex locked)
func (wsc *WebSocketClient) reconnect(ctx context.Context) error {
	if !wsc.started.Load() {
		return nil
	}
	// Skip reconnect if context is already done
	if ctx.Err() != nil {
		return nil
	}

	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if wsc.client != nil {
		wsc.client.Stop()
		wsc.client = nil
	}
	// Create a new SimpleWebSocketClient
	wsc.client = NewSimpleWebSocketClient(
		wsc.wsURL,
		// Message handler
		func(message []byte) {
			if err := wsc.onMessage(message); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		},
		// Error handler - handles reconnection
		func(err error) {
			log.Printf("WebSocket error: %v", err)
			go wsc.reconnect(ctx)
		},
	)

	// Start the client
	wsc.client.Start(ctx)

	return nil
}

// Close closes the WebSocket connection
// It's safe to call this method multiple times
func (wsc *WebSocketClient) Close() {
	wsc.mu.Lock()

	if wsc.client != nil {
		clientToStop := wsc.client
		wsc.client = nil
		wsc.mu.Unlock()

		// Stop the client outside the mutex lock to avoid potential deadlocks
		clientToStop.Stop()
	} else {
		wsc.mu.Unlock()
	}
}
