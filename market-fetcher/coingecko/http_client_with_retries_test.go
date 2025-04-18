package coingecko

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPClientWithRetries_Timeouts tests that the client correctly applies timeouts
func TestHTTPClientWithRetries_Timeouts(t *testing.T) {
	// Create a test server that sleeps to simulate slow responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delay := r.URL.Query().Get("delay")
		if delay == "connection" {
			// This won't actually test connection timeout in unit tests
			// since we're using httptest, but it's included for completeness
			time.Sleep(500 * time.Millisecond)
		} else if delay == "response" {
			// Simulate a slow response
			time.Sleep(500 * time.Millisecond)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Test with short timeout
	t.Run("RequestTimeout", func(t *testing.T) {
		opts := DefaultRetryOptions()
		opts.RequestTimeout = 100 * time.Millisecond // Very short timeout

		client := NewHTTPClientWithRetries(opts)

		req, _ := http.NewRequest("GET", server.URL+"?delay=response", nil)
		_, _, _, err := client.ExecuteRequest(req)

		if err == nil {
			t.Error("Expected timeout error, got none")
		}
	})

	// Test with sufficient timeout
	t.Run("NoTimeout", func(t *testing.T) {
		opts := DefaultRetryOptions()
		opts.RequestTimeout = 2 * time.Second // Sufficient timeout

		client := NewHTTPClientWithRetries(opts)

		req, _ := http.NewRequest("GET", server.URL, nil)
		_, _, _, err := client.ExecuteRequest(req)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

// TestHTTPClientWithRetries_Retries tests the retry behavior
func TestHTTPClientWithRetries_Retries(t *testing.T) {
	// Track request attempts
	attempts := 0

	// Create a test server that fails initially and succeeds after retries
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		// Fail the first two attempts
		if attempts <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable) // 503 Service Unavailable
			w.Write([]byte(`{"error":"service unavailable"}`))
			return
		}

		// Succeed on the third attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Configure client with 3 retries and minimal backoff
	opts := DefaultRetryOptions()
	opts.MaxRetries = 3
	opts.BaseBackoff = 10 * time.Millisecond // Minimal backoff for tests

	client := NewHTTPClientWithRetries(opts)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, body, duration, err := client.ExecuteRequest(req)

	// Check results
	if err != nil {
		t.Errorf("Expected successful request after retries, got error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if string(body) != `{"status":"ok"}` {
		t.Errorf("Expected body '{\"status\":\"ok\"}', got '%s'", string(body))
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if duration <= 0 {
		t.Errorf("Expected positive duration, got %v", duration)
	}
}

// TestHTTPClientWithRetries_MaxRetriesExceeded tests behavior when max retries are exceeded
func TestHTTPClientWithRetries_MaxRetriesExceeded(t *testing.T) {
	// Track request attempts
	attempts := 0

	// Create a test server that always fails with 503
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable) // 503 Service Unavailable
		w.Write([]byte(`{"error":"service unavailable"}`))
	}))
	defer server.Close()

	// Configure client with 2 retries and minimal backoff
	opts := DefaultRetryOptions()
	opts.MaxRetries = 2
	opts.BaseBackoff = 10 * time.Millisecond // Minimal backoff for tests

	client := NewHTTPClientWithRetries(opts)

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, _, _, err := client.ExecuteRequest(req)

	// Should get an error after all retries fail
	if err == nil {
		t.Error("Expected error after exceeding max retries, got none")
	}

	// Should have attempted exactly MaxRetries times
	if attempts != opts.MaxRetries {
		t.Errorf("Expected %d attempts, got %d", opts.MaxRetries, attempts)
	}
}

// TestHTTPClientWithRetries_NonRetryableError tests that non-retryable errors fail immediately
func TestHTTPClientWithRetries_NonRetryableError(t *testing.T) {
	// Track request attempts
	attempts := 0

	// Create a test server that returns a non-retryable error (400 Bad Request)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest) // 400 Bad Request - non-retryable
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	// Configure client with retries
	opts := DefaultRetryOptions()
	opts.MaxRetries = 3

	client := NewHTTPClientWithRetries(opts)

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, _, _, err := client.ExecuteRequest(req)

	// Should get an error
	if err == nil {
		t.Error("Expected error for non-retryable status code, got none")
	}

	// Should have attempted only once
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

// TestHTTPClientWithRetries_ConnectionFailure tests handling of connection failures
func TestHTTPClientWithRetries_ConnectionFailure(t *testing.T) {
	// Create a client pointing to a non-existent server
	opts := DefaultRetryOptions()
	opts.MaxRetries = 2
	opts.BaseBackoff = 10 * time.Millisecond
	opts.ConnectionTimeout = 100 * time.Millisecond

	client := NewHTTPClientWithRetries(opts)

	// Point to a localhost address that should immediately fail to connect
	req, _ := http.NewRequest("GET", "http://localhost:57891", nil) // Using an unlikely port

	startTime := time.Now()
	_, _, _, err := client.ExecuteRequest(req)
	duration := time.Since(startTime)

	// Should get an error
	if err == nil {
		t.Error("Expected error for connection failure, got none")
	}

	// The total duration should be less than what would be required for the full
	// MaxRetries × (ConnectionTimeout + BaseBackoff × 2^retry) if timeouts weren't working
	maxExpectedDuration := time.Duration(opts.MaxRetries) * (opts.ConnectionTimeout + opts.BaseBackoff*3)
	if duration > maxExpectedDuration {
		t.Errorf("Request took longer than expected: %v > %v", duration, maxExpectedDuration)
	}
}

// mockTransport is a mock http.RoundTripper for testing custom behavior
type mockTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

// TestHTTPClientWithRetries_NetworkErrors tests handling of various network errors
func TestHTTPClientWithRetries_NetworkErrors(t *testing.T) {
	// Mock transport that returns network errors
	attempts := 0
	transport := &mockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return nil, errors.New("connection reset by peer")
			} else {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}
		},
	}

	// Configure client with custom transport
	opts := DefaultRetryOptions()
	opts.MaxRetries = 2
	opts.BaseBackoff = 10 * time.Millisecond

	client := &HTTPClientWithRetries{
		client: &http.Client{Transport: transport},
		opts:   opts,
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, body, _, err := client.ExecuteRequest(req)

	// Should succeed after retry
	if err != nil {
		t.Errorf("Expected success after retry, got error: %v", err)
	}

	if string(body) != `{"status":"ok"}` {
		t.Errorf("Expected body '{\"status\":\"ok\"}', got '%s'", string(body))
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}
