package binance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketCallback is a callback function for handling WebSocket messages
type WebSocketCallback func(message []byte)

// ErrorCallback is a callback function for handling WebSocket errors
type ErrorCallback func(err error)

// SimpleWebSocketClient is a simplified WebSocket client
type SimpleWebSocketClient struct {
	// WebSocket URL
	wsURL string

	// WebSocket connection
	conn *websocket.Conn

	// Callback for handling messages
	onMessage WebSocketCallback

	// Callback for handling errors
	onError ErrorCallback

	// WaitGroup for the message loop
	loopWg sync.WaitGroup

	// Context cancel function
	cancelFunc context.CancelFunc
}

// NewSimpleWebSocketClient creates a new simple WebSocket client
func NewSimpleWebSocketClient(wsURL string, onMessage WebSocketCallback, onError ErrorCallback) *SimpleWebSocketClient {
	if wsURL == "" {
		wsURL = BASE_WS_URL
	}

	return &SimpleWebSocketClient{
		wsURL:     wsURL,
		onMessage: onMessage,
		onError:   onError,
	}
}

// Start establishes connection to WebSocket server and starts the message loop
func (c *SimpleWebSocketClient) Start(ctx context.Context) {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel

	// Connect to WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(c.wsURL, nil)
	if err != nil {
		c.onError(fmt.Errorf("failed to connect to WebSocket: %v", err))
		return
	}

	// Store the connection
	c.conn = conn

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(PONG_TIMEOUT)); err != nil {
		c.onError(fmt.Errorf("failed to set read deadline: %v", err))
		return
	}

	// Setup ping/pong handlers
	c.setupPingPong()

	// Start message loop
	c.startMessageLoop(ctx)
}

// Stop stops the message loop and closes the connection
// It blocks until the message loop is completely terminated
func (c *SimpleWebSocketClient) Stop() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	if c.conn != nil {
		c.conn.Close()
	}

	// Wait for message loop to exit
	c.loopWg.Wait()
}

// setupPingPong sets up ping/pong handlers
func (c *SimpleWebSocketClient) setupPingPong() {
	// Set ping handler to respond with pong containing the same data
	c.conn.SetPingHandler(func(appData string) error {
		// Reset read deadline
		if err := c.conn.SetReadDeadline(time.Now().Add(PONG_TIMEOUT)); err != nil {
			return fmt.Errorf("failed to set read deadline in ping handler: %v", err)
		}
		// Respond with pong containing the same data
		err := c.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
		if err != nil {
			c.onError(fmt.Errorf("error sending pong response: %v", err))
		}
		return nil
	})
}

// startMessageLoop begins reading messages from the WebSocket connection
func (c *SimpleWebSocketClient) startMessageLoop(ctx context.Context) {
	// Add to wait group before starting the loop
	c.loopWg.Add(1)

	// Start message loop in a goroutine
	go func() {
		// Mark the loop as done when exiting
		defer c.loopWg.Done()

		for {
			select {
			case <-ctx.Done():
				// Context was cancelled, exit the loop
				return
			default:
				// Read message with timeout
				if err := c.conn.SetReadDeadline(time.Now().Add(PONG_TIMEOUT)); err != nil {
					c.onError(fmt.Errorf("failed to set read deadline in message loop: %v", err))
					return
				}
				_, message, err := c.conn.ReadMessage()

				if err != nil {
					c.onError(fmt.Errorf("error reading WebSocket message: %v", err))
					return
				}

				// Process message with the provided handler
				c.onMessage(message)
			}
		}
	}()
}
