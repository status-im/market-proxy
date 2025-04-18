package binance

import (
	"context"
	"fmt"
	"log"
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
	conn      *websocket.Conn
	stopCh    chan struct{}
	onMessage func(message []byte) error
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(stopCh chan struct{}, onMessage func(message []byte) error) *WebSocketClient {
	return &WebSocketClient{
		stopCh:    stopCh,
		onMessage: onMessage,
	}
}

// Connect establishes connection to Binance WebSocket API
func (wsc *WebSocketClient) Connect() error {
	// Connect to Binance WebSocket data stream
	conn, _, err := websocket.DefaultDialer.Dial(BASE_WS_URL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Binance WebSocket: %v", err)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(PONG_TIMEOUT))

	wsc.conn = conn
	return nil
}

// SetupPingPong sets up ping/pong handlers for the WebSocket connection
func (wsc *WebSocketClient) SetupPingPong() {
	// Start ping handler
	go func() {
		ticker := time.NewTicker(PING_INTERVAL)
		defer ticker.Stop()

		for {
			select {
			case <-wsc.stopCh:
				return
			case <-ticker.C:
				// Send empty pong frame
				if err := wsc.conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(PING_INTERVAL)); err != nil {
					log.Printf("Error sending pong: %v", err)
					return
				}
			}
		}
	}()

	// Set ping handler
	wsc.conn.SetPingHandler(func(string) error {
		// Reset read deadline
		wsc.conn.SetReadDeadline(time.Now().Add(PONG_TIMEOUT))
		return nil
	})
}

// StartMessageLoop begins reading messages from the WebSocket connection
func (wsc *WebSocketClient) StartMessageLoop(ctx context.Context, reconnectFn func()) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-wsc.stopCh:
			return
		default:
			// Read message
			_, message, err := wsc.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Error reading WebSocket message: %v", err)
				}
				// Try to reconnect
				reconnectFn()
				continue
			}

			// Process message with the provided handler
			if err := wsc.onMessage(message); err != nil {
				log.Printf("%v", err)
			}
		}
	}
}

// Close closes the WebSocket connection
func (wsc *WebSocketClient) Close() {
	if wsc.conn != nil {
		wsc.conn.Close()
		wsc.conn = nil
	}
}
