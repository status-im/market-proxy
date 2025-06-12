package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestSimpleWebSocketClient_Start_Stop tests that the client can be started and stopped
func TestSimpleWebSocketClient_Start_Stop(t *testing.T) {
	// Create a test server
	server, wsURL := createTestServer(t, func(conn *websocket.Conn) {
		// Just echo messages back
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	})
	defer server.Close()

	// Message and error channels
	receivedMessages := make(chan []byte, 10)
	receivedErrors := make(chan error, 10)

	// Create client
	client := NewSimpleWebSocketClient(
		wsURL,
		func(message []byte) { receivedMessages <- message },
		func(err error) {
			// We expect a connection closed error when stopping the client
			if websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) ||
				strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			receivedErrors <- err
		},
	)

	// Start client
	ctx := context.Background()
	client.Start(ctx)

	// Check for any connection errors
	select {
	case err := <-receivedErrors:
		t.Fatalf("Error starting client: %v", err)
	case <-time.After(100 * time.Millisecond):
		// No errors, proceed
	}

	// Stop client after a short delay
	time.Sleep(100 * time.Millisecond)
	client.Stop()

	// Verify client stopped properly
	select {
	case <-time.After(100 * time.Millisecond):
		// This is good, no errors reported
	case err := <-receivedErrors:
		t.Fatalf("Unexpected error: %v", err)
	}
}

// TestSimpleWebSocketClient_MessageHandling tests that messages are properly delivered to the handler
func TestSimpleWebSocketClient_MessageHandling(t *testing.T) {
	testMessage := []byte("test message")
	messageReceived := make(chan struct{})

	// Create a test server that sends one message and then waits
	server, wsURL := createTestServer(t, func(conn *websocket.Conn) {
		// Send a test message
		if err := conn.WriteMessage(websocket.TextMessage, testMessage); err != nil {
			return
		}

		// Wait for test to finish
		<-time.After(500 * time.Millisecond)
	})
	defer server.Close()

	// Create client with handlers
	client := NewSimpleWebSocketClient(
		wsURL,
		func(message []byte) {
			if string(message) == string(testMessage) {
				close(messageReceived)
			}
		},
		func(err error) {
			// Ignore expected connection close errors
			if websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) ||
				strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			t.Errorf("Unexpected error: %v", err)
		},
	)

	// Start client
	ctx := context.Background()
	client.Start(ctx)
	defer client.Stop()

	// Wait for message to be received
	select {
	case <-messageReceived:
		// Success, message was received
	case <-time.After(1 * time.Second):
		t.Fatalf("Timed out waiting for message")
	}
}

// TestSimpleWebSocketClient_ErrorHandling tests that errors are properly delivered to the handler
func TestSimpleWebSocketClient_ErrorHandling(t *testing.T) {
	errorReceived := make(chan struct{})

	// Create a test server that closes the connection immediately
	server, wsURL := createTestServer(t, func(conn *websocket.Conn) {
		conn.Close()
	})
	defer server.Close()

	// Create client with handlers
	client := NewSimpleWebSocketClient(
		wsURL,
		func(message []byte) {
			t.Errorf("Unexpected message: %s", string(message))
		},
		func(err error) {
			close(errorReceived)
		},
	)

	// Start client
	ctx := context.Background()
	client.Start(ctx)
	defer client.Stop()

	// Wait for error to be received
	select {
	case <-errorReceived:
		// Success, error was received
	case <-time.After(1 * time.Second):
		t.Fatalf("Timed out waiting for error")
	}
}

// TestSimpleWebSocketClient_Stop_WaitsForLoop tests that Stop waits for the message loop to exit
func TestSimpleWebSocketClient_Stop_WaitsForLoop(t *testing.T) {
	// Create a signal channel to track message processing
	messageProcessed := make(chan struct{})

	// Create a test server with controlled message flow
	server, wsURL := createTestServer(t, func(conn *websocket.Conn) {
		// Send a single message and wait
		if err := conn.WriteMessage(websocket.TextMessage, []byte("test message")); err != nil {
			return
		}

		// Wait for test to signal completion
		select {
		case <-messageProcessed:
			// Test is done, we can exit
		case <-time.After(2 * time.Second):
			// Safety timeout
		}
	})
	defer server.Close()

	// Track when the message handler is called
	messageHandlerStarted := make(chan struct{})
	messageHandlerCompleted := make(chan struct{})

	// Create client with a handler that signals when it starts and completes
	client := NewSimpleWebSocketClient(
		wsURL,
		func(message []byte) {
			// Signal that message handler has started
			close(messageHandlerStarted)

			// Simulate work that takes time
			time.Sleep(200 * time.Millisecond)

			// Signal that the handler is done
			close(messageHandlerCompleted)
		},
		func(err error) {
			// Ignore expected connection close errors
			if websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) ||
				strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			t.Errorf("Unexpected error: %v", err)
		},
	)

	// Start client
	ctx := context.Background()
	client.Start(ctx)

	// Wait for the message handler to start
	select {
	case <-messageHandlerStarted:
		// Handler has started, continue
	case <-time.After(1 * time.Second):
		t.Fatalf("Timed out waiting for message handler to start")
	}

	// Stop client - this should wait for the message handler to complete
	stopStart := time.Now()
	go func() {
		client.Stop()
		close(messageProcessed) // Signal the test server that we're done
	}()

	// Check if the handler completes before Stop returns
	select {
	case <-messageHandlerCompleted:
		// Handler completed, good
		stopDuration := time.Since(stopStart)

		// Stop should have taken at least as long as the handler sleeps
		if stopDuration < 150*time.Millisecond {
			t.Errorf("Stop appears to have returned too quickly (%v), should have waited for message handler", stopDuration)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("Timed out waiting for message handler to complete")
	}
}

// TestSimpleWebSocketClient_ConnectionError tests that connection errors are reported via onError
func TestSimpleWebSocketClient_ConnectionError(t *testing.T) {
	// Error channel to capture errors
	errorReceived := make(chan error, 1)

	// Create client with an invalid URL
	client := NewSimpleWebSocketClient(
		"ws://invalid-host-that-does-not-exist.example",
		func(message []byte) {
			t.Errorf("Unexpected message: %s", string(message))
		},
		func(err error) {
			errorReceived <- err
		},
	)

	// Start client - this should fail to connect
	ctx := context.Background()
	client.Start(ctx)
	defer client.Stop()

	// Wait for error to be received
	select {
	case err := <-errorReceived:
		// Success, error was received
		if err == nil {
			t.Fatalf("Expected error but got nil")
		}
		// The error should contain something about connection or dial
		if !strings.Contains(err.Error(), "connect") &&
			!strings.Contains(err.Error(), "dial") &&
			!strings.Contains(err.Error(), "lookup") {
			t.Fatalf("Expected connection error but got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("Timed out waiting for connection error")
	}
}

// createTestServer creates a test WebSocket server
func createTestServer(t *testing.T, handler func(*websocket.Conn)) (*httptest.Server, string) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		handler(conn)
	}))

	// Convert http:// to ws://
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)
	return server, wsURL
}
