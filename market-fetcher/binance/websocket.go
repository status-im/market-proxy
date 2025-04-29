package binance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	BASE_WS_URL = "wss://data-stream.binance.vision/ws/!ticker@arr"
	// Connection timeouts
	PING_INTERVAL = 20 * time.Second
	PONG_TIMEOUT  = 60 * time.Second
)

// WebSocketClient manages WebSocket connection to Binance
type WebSocketClient struct {
	conn       *websocket.Conn
	onMessage  func(message []byte) error
	wsURL      string
	mu         sync.Mutex
	isRunning  bool
	cancelFunc context.CancelFunc
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(onMessage func(message []byte) error, wsURL string) *WebSocketClient {
	if wsURL == "" {
		wsURL = BASE_WS_URL
	}
	return &WebSocketClient{
		onMessage: onMessage,
		wsURL:     wsURL,
		isRunning: false,
	}
}

// Connect establishes connection to Binance WebSocket API with the given context
// It's safe to call this method multiple times - it will only connect once unless Close() was called
func (wsc *WebSocketClient) Connect(ctx context.Context) error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	// Don't reconnect if already connected and running
	if wsc.isConnected() {
		return nil
	}

	connectCtx, cancel := context.WithCancel(ctx)
	wsc.cancelFunc = cancel

	return wsc.connect(connectCtx)
}

// connect establishes a new connection (must be called with mutex locked)
func (wsc *WebSocketClient) connect(ctx context.Context) error {
	// Connect to Binance WebSocket data stream
	conn, _, err := websocket.DefaultDialer.Dial(wsc.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Binance WebSocket: %v", err)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(PONG_TIMEOUT))

	wsc.conn = conn

	// Setup ping/pong handlers
	wsc.setupPingPong()

	// Start message loop
	wsc.isRunning = true
	go wsc.startMessageLoop(ctx)

	return nil
}

// closeAndConnect closes the current connection and attempts to reconnect
func (wsc *WebSocketClient) closeAndConnect(ctx context.Context) {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if wsc.cancelFunc != nil {
		wsc.cancelFunc()
	}

	// Close connection but don't set isRunning to false
	if wsc.conn != nil {
		wsc.conn.Close()
		wsc.conn = nil
	}

	// Only attempt to reconnect if we're supposed to be running
	if !wsc.isRunning {
		return
	}

	connectCtx, cancel := context.WithCancel(ctx)
	wsc.cancelFunc = cancel

	// Attempt to reconnect
	if err := wsc.connect(connectCtx); err != nil {
		log.Printf("Failed to reconnect to Binance WebSocket: %v", err)
	}
}

// Close closes the WebSocket connection
// It's safe to call this method multiple times
func (wsc *WebSocketClient) Close() {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if !wsc.isRunning {
		return
	}

	if wsc.cancelFunc != nil {
		wsc.cancelFunc()
		wsc.cancelFunc = nil
	}

	// Close connection
	if wsc.conn != nil {
		wsc.conn.Close()
		wsc.conn = nil
	}

	wsc.isRunning = false
}

// isConnected returns true if the client is connected
func (wsc *WebSocketClient) isConnected() bool {
	return wsc.isRunning && wsc.conn != nil
}

// setupPingPong sets up ping/pong handlers for the WebSocket connection
func (wsc *WebSocketClient) setupPingPong() {
	// Set ping handler to respond with pong containing the same data
	wsc.conn.SetPingHandler(func(appData string) error {
		// Reset read deadline
		wsc.conn.SetReadDeadline(time.Now().Add(PONG_TIMEOUT))
		// Respond with pong containing the same data
		err := wsc.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
		if err != nil {
			log.Printf("Error sending pong response: %v", err)
		}
		return nil
	})
}

// startMessageLoop begins reading messages from the WebSocket connection
func (wsc *WebSocketClient) startMessageLoop(ctx context.Context) {
	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			wsc.conn.Close()
			close(done)
			return
		}
	}()

	for {
		select {
		case <-done:
			return
		default:
			if !wsc.isConnected() {
				return
			}

			// Read message
			_, message, err := wsc.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Error reading WebSocket message: %v", err)
				}

				// Try to reconnect
				go wsc.closeAndConnect(ctx)
				return
			}

			// Process message with the provided handler
			if err := wsc.onMessage(message); err != nil {
				log.Printf("%v", err)
			}
		}
	}
}
