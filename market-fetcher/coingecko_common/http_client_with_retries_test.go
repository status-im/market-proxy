package coingecko_common

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockHttpStatusHandler implements IHttpStatusHandler for testing
type MockHttpStatusHandler struct {
	requestStatuses []string
	retryCount      int
}

func NewMockHttpStatusHandler() *MockHttpStatusHandler {
	return &MockHttpStatusHandler{
		requestStatuses: make([]string, 0),
		retryCount:      0,
	}
}

func (m *MockHttpStatusHandler) OnRequest(status string) {
	m.requestStatuses = append(m.requestStatuses, status)
}

func (m *MockHttpStatusHandler) OnRetry() {
	m.retryCount++
}

// GetRequestCount returns the number of requests recorded
func (m *MockHttpStatusHandler) GetRequestCount() int {
	return len(m.requestStatuses)
}

// GetRetryCount returns the number of retries recorded
func (m *MockHttpStatusHandler) GetRetryCount() int {
	return m.retryCount
}

// TestHTTPClientWithRetries_Timeouts tests that the Client correctly applies timeouts
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
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			return
		}
	}))
	defer server.Close()

	// Test with short timeout
	t.Run("RequestTimeout", func(t *testing.T) {
		opts := DefaultRetryOptions()
		opts.RequestTimeout = 100 * time.Millisecond // Very short timeout
		opts.MaxRetries = 3

		mockHandler := NewMockHttpStatusHandler()
		client := NewHTTPClientWithRetries(opts, mockHandler)

		req, _ := http.NewRequest("GET", server.URL+"?delay=response", nil)
		_, _, _, err := client.ExecuteRequest(req)

		if err == nil {
			t.Error("Expected timeout error, got none")
		}

		// Should have recorded errors
		if mockHandler.GetRequestCount() < 1 {
			t.Errorf("Expected at least 1 request status, got %d", mockHandler.GetRequestCount())
		}

		// Should have recorded retries
		if mockHandler.GetRetryCount() < 1 {
			t.Errorf("Expected at least 1 retry, got %d", mockHandler.GetRetryCount())
		}
	})

	// Test with sufficient timeout
	t.Run("NoTimeout", func(t *testing.T) {
		opts := DefaultRetryOptions()
		opts.RequestTimeout = 2 * time.Second // Sufficient timeout

		mockHandler := NewMockHttpStatusHandler()
		client := NewHTTPClientWithRetries(opts, mockHandler)

		req, _ := http.NewRequest("GET", server.URL, nil)
		_, _, _, err := client.ExecuteRequest(req)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Should have recorded a success
		if mockHandler.GetRequestCount() != 1 || mockHandler.requestStatuses[0] != "success" {
			t.Errorf("Expected 1 success status, got %v", mockHandler.requestStatuses)
		}

		// Should not have recorded retries
		if mockHandler.GetRetryCount() != 0 {
			t.Errorf("Expected 0 retries, got %d", mockHandler.GetRetryCount())
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
			if _, err := w.Write([]byte(`{"error":"service unavailable"}`)); err != nil {
				return
			}
			return
		}

		// Succeed on the third attempt
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			return
		}
	}))
	defer server.Close()

	// Configure Client with 3 retries and minimal backoff
	opts := DefaultRetryOptions()
	opts.MaxRetries = 3
	opts.BaseBackoff = 10 * time.Millisecond // Minimal backoff for tests

	mockHandler := NewMockHttpStatusHandler()
	client := NewHTTPClientWithRetries(opts, mockHandler)

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

	// Should have recorded 2 rate_limited and 1 success
	if mockHandler.GetRequestCount() != 3 ||
		mockHandler.requestStatuses[0] != "rate_limited" ||
		mockHandler.requestStatuses[1] != "rate_limited" ||
		mockHandler.requestStatuses[2] != "success" {
		t.Errorf("Expected [rate_limited, rate_limited, success] statuses, got %v", mockHandler.requestStatuses)
	}

	// Should have recorded 2 retries
	if mockHandler.GetRetryCount() != 2 {
		t.Errorf("Expected 2 retries, got %d", mockHandler.GetRetryCount())
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
		if _, err := w.Write([]byte(`{"error":"service unavailable"}`)); err != nil {
			return
		}
	}))
	defer server.Close()

	// Configure Client with 2 retries and minimal backoff
	opts := DefaultRetryOptions()
	opts.MaxRetries = 2
	opts.BaseBackoff = 10 * time.Millisecond // Minimal backoff for tests

	mockHandler := NewMockHttpStatusHandler()
	client := NewHTTPClientWithRetries(opts, mockHandler)

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

	// Should have recorded rate_limited for each attempt
	if mockHandler.GetRequestCount() != opts.MaxRetries {
		t.Errorf("Expected %d request statuses, got %d", opts.MaxRetries, mockHandler.GetRequestCount())
	}

	// Should have recorded retries
	if mockHandler.GetRetryCount() != opts.MaxRetries-1 {
		t.Errorf("Expected %d retries, got %d", opts.MaxRetries-1, mockHandler.GetRetryCount())
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
		if _, err := w.Write([]byte(`{"error":"bad request"}`)); err != nil {
			return
		}
	}))
	defer server.Close()

	// Configure Client with retries
	opts := DefaultRetryOptions()
	opts.MaxRetries = 3

	mockHandler := NewMockHttpStatusHandler()
	client := NewHTTPClientWithRetries(opts, mockHandler)

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

	// Should have recorded error without retry
	if mockHandler.GetRequestCount() != 1 || mockHandler.requestStatuses[0] != "error" {
		t.Errorf("Expected [error] status, got %v", mockHandler.requestStatuses)
	}

	// Should not have recorded retries
	if mockHandler.GetRetryCount() != 0 {
		t.Errorf("Expected 0 retries, got %d", mockHandler.GetRetryCount())
	}
}

// TestHTTPClientWithRetries_ConnectionFailure tests handling of connection failures
func TestHTTPClientWithRetries_ConnectionFailure(t *testing.T) {
	// Create a Client pointing to a non-existent server
	opts := DefaultRetryOptions()
	opts.MaxRetries = 2
	opts.BaseBackoff = 10 * time.Millisecond
	opts.ConnectionTimeout = 100 * time.Millisecond

	mockHandler := NewMockHttpStatusHandler()
	client := NewHTTPClientWithRetries(opts, mockHandler)

	// Point to a localhost address that should immediately fail to connect
	req, _ := http.NewRequest("GET", "http://localhost:57891", nil) // Using an unlikely port

	startTime := time.Now()
	_, _, _, err := client.ExecuteRequest(req)
	duration := time.Since(startTime)

	// Should get an error
	if err == nil {
		t.Error("Expected connection error, got none")
	}

	// Should fail quickly
	maxExpectedDuration := time.Duration(opts.MaxRetries) * (opts.ConnectionTimeout + opts.BaseBackoff*2)
	if duration > maxExpectedDuration {
		t.Errorf("Expected quick failure, took %v (exceeds %v)", duration, maxExpectedDuration)
	}

	// Should have recorded errors
	if mockHandler.GetRequestCount() != opts.MaxRetries {
		t.Errorf("Expected %d request statuses, got %d", opts.MaxRetries, mockHandler.GetRequestCount())
	}

	// Should have recorded retries
	if mockHandler.GetRetryCount() != opts.MaxRetries-1 {
		t.Errorf("Expected %d retries, got %d", opts.MaxRetries-1, mockHandler.GetRetryCount())
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
	// Configure Client with retries
	opts := DefaultRetryOptions()
	opts.MaxRetries = 2
	opts.BaseBackoff = 10 * time.Millisecond

	mockHandler := NewMockHttpStatusHandler()
	client := NewHTTPClientWithRetries(opts, mockHandler)

	// Replace the Client's transport with our mock
	errorReturned := false
	client.Client.Transport = &mockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			if !errorReturned {
				// First request fails with connection reset
				errorReturned = true
				return nil, errors.New("connection reset by peer")
			}
			// Subsequent requests succeed
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, body, _, err := client.ExecuteRequest(req)

	// Should not get an error
	if err != nil {
		t.Errorf("Expected success after retrying network error, got: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if string(body) != `{"status":"ok"}` {
		t.Errorf("Expected body '{\"status\":\"ok\"}', got '%s'", string(body))
	}

	// Should have recorded error followed by success
	if mockHandler.GetRequestCount() != 2 ||
		mockHandler.requestStatuses[0] != "error" ||
		mockHandler.requestStatuses[1] != "success" {
		t.Errorf("Expected [error, success] statuses, got %v", mockHandler.requestStatuses)
	}

	// Should have recorded 1 retry
	if mockHandler.GetRetryCount() != 1 {
		t.Errorf("Expected 1 retry, got %d", mockHandler.GetRetryCount())
	}
}

// Test with no metrics handler
func TestHTTPClientWithRetries_NoHandler(t *testing.T) {
	// Create a server that always succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			return
		}
	}))
	defer server.Close()

	// Configure Client with null handler
	opts := DefaultRetryOptions()
	client := NewHTTPClientWithRetries(opts, nil)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, body, _, err := client.ExecuteRequest(req)

	// Should succeed without error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if string(body) != `{"status":"ok"}` {
		t.Errorf("Expected body '{\"status\":\"ok\"}', got '%s'", string(body))
	}
}
