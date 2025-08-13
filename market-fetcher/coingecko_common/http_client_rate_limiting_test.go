package coingecko_common

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mock_coingecko_common "github.com/status-im/market-proxy/coingecko_common/mocks"
	"go.uber.org/mock/gomock"
	"golang.org/x/time/rate"
)

func TestHTTPClientWithRetries_RateLimiting_NoLimiter(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create mock manager that returns no limiter
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockManager := mock_coingecko_common.NewMockIRateLimiterManager(ctrl)

	// Expect GetLimiterForURL to be called and return nil (no limiter)
	mockManager.EXPECT().GetLimiterForURL(gomock.Any()).Return(nil)

	opts := DefaultRetryOptions()
	opts.MaxRetries = 1 // Single attempt for faster test

	client := NewHTTPClientWithRetries(opts, nil, mockManager)

	req, _ := http.NewRequest("GET", server.URL, nil)
	start := time.Now()

	_, _, _, err := client.ExecuteRequest(req)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should complete quickly since no rate limiting
	if duration > 100*time.Millisecond {
		t.Errorf("Expected quick completion, took %v", duration)
	}
}

func TestHTTPClientWithRetries_RateLimiting_WithLimiter(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create mock manager with a strict rate limiter
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockManager := mock_coingecko_common.NewMockIRateLimiterManager(ctrl)

	// Very restrictive limiter: 1 request per 2 seconds, burst of 1
	limiter := rate.NewLimiter(rate.Every(2*time.Second), 1)

	// Expect GetLimiterForURL to be called twice and return the limiter
	mockManager.EXPECT().GetLimiterForURL(gomock.Any()).Return(limiter).Times(2)

	opts := DefaultRetryOptions()
	opts.MaxRetries = 1

	client := NewHTTPClientWithRetries(opts, nil, mockManager)

	req, _ := http.NewRequest("GET", server.URL, nil)

	// First request should succeed quickly
	start1 := time.Now()
	_, _, _, err1 := client.ExecuteRequest(req)
	duration1 := time.Since(start1)

	if err1 != nil {
		t.Errorf("Expected no error on first request, got: %v", err1)
	}

	if duration1 > 100*time.Millisecond {
		t.Errorf("Expected first request to be quick, took %v", duration1)
	}

	// Second request should be delayed by rate limiter
	start2 := time.Now()
	_, _, _, err2 := client.ExecuteRequest(req)
	duration2 := time.Since(start2)

	if err2 != nil {
		t.Errorf("Expected no error on second request, got: %v", err2)
	}

	// Should take at least close to 2 seconds due to rate limiting
	if duration2 < 1500*time.Millisecond {
		t.Errorf("Expected second request to be rate limited (>1.5s), took %v", duration2)
	}
}

func TestHTTPClientWithRetries_RateLimiting_ContextCancellation(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create mock manager with a very slow rate limiter
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockManager := mock_coingecko_common.NewMockIRateLimiterManager(ctrl)

	limiter := rate.NewLimiter(rate.Every(10*time.Second), 0) // Very slow, no burst

	// Expect GetLimiterForURL to be called and return the slow limiter
	mockManager.EXPECT().GetLimiterForURL(gomock.Any()).Return(limiter)

	opts := DefaultRetryOptions()
	opts.MaxRetries = 1

	client := NewHTTPClientWithRetries(opts, nil, mockManager)

	// Create request with short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)

	start := time.Now()
	_, _, _, err := client.ExecuteRequest(req)
	duration := time.Since(start)

	// Should fail due to context cancellation during rate limiting wait
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	// Should complete quickly due to context timeout
	if duration > 200*time.Millisecond {
		t.Errorf("Expected quick failure due to context timeout, took %v", duration)
	}

	// Error should mention rate limiter wait failure
	if err != nil && err.Error() != "" {
		// Just verify we got an error - the exact message may vary
		t.Logf("Got expected error: %v", err)
	}
}

func TestHTTPClientWithRetries_RateLimiting_NilManager(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	opts := DefaultRetryOptions()
	opts.MaxRetries = 1

	// Create client with nil manager
	client := NewHTTPClientWithRetries(opts, nil, nil)

	req, _ := http.NewRequest("GET", server.URL, nil)
	start := time.Now()

	_, _, _, err := client.ExecuteRequest(req)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error with nil manager, got: %v", err)
	}

	// Should complete quickly since no rate limiting
	if duration > 100*time.Millisecond {
		t.Errorf("Expected quick completion with nil manager, took %v", duration)
	}
}

func TestHTTPClientWithRetries_RateLimiting_MultipleRequests(t *testing.T) {
	// Create a test server that counts requests
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create mock manager with moderate rate limiter
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockManager := mock_coingecko_common.NewMockIRateLimiterManager(ctrl)

	// 2 requests per second, burst of 2
	limiter := rate.NewLimiter(2, 2)

	// Expect GetLimiterForURL to be called 3 times and return the limiter
	mockManager.EXPECT().GetLimiterForURL(gomock.Any()).Return(limiter).Times(3)

	opts := DefaultRetryOptions()
	opts.MaxRetries = 1

	client := NewHTTPClientWithRetries(opts, nil, mockManager)

	req, _ := http.NewRequest("GET", server.URL, nil)

	start := time.Now()

	// Make 3 requests quickly
	for i := 0; i < 3; i++ {
		_, _, _, err := client.ExecuteRequest(req)
		if err != nil {
			t.Errorf("Request %d failed: %v", i+1, err)
		}
	}

	duration := time.Since(start)

	// First 2 should be quick (burst), 3rd should be rate limited
	if duration < 400*time.Millisecond {
		t.Errorf("Expected some rate limiting delay, took only %v", duration)
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests to reach server, got %d", requestCount)
	}
}

func TestHTTPClientWithRetries_RateLimiting_WithRetries(t *testing.T) {
	// Create a test server that fails first request, succeeds on retry
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, err := w.Write([]byte(`{"error":"service unavailable"}`)); err != nil {
				t.Errorf("Failed to write error response: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
				t.Errorf("Failed to write success response: %v", err)
			}
		}
	}))
	defer server.Close()

	// Create mock manager with rate limiter
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockManager := mock_coingecko_common.NewMockIRateLimiterManager(ctrl)

	limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 1)

	// Expect GetLimiterForURL to be called twice (once per attempt) and return the limiter
	mockManager.EXPECT().GetLimiterForURL(gomock.Any()).Return(limiter).Times(2)

	opts := DefaultRetryOptions()
	opts.MaxRetries = 2
	opts.BaseBackoff = 10 * time.Millisecond // Small backoff for faster test

	client := NewHTTPClientWithRetries(opts, nil, mockManager)

	req, _ := http.NewRequest("GET", server.URL, nil)
	start := time.Now()

	_, _, _, err := client.ExecuteRequest(req)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Expected success after retry, got: %v", err)
	}

	// Should take some time due to rate limiting on both attempts
	if duration < 100*time.Millisecond {
		t.Errorf("Expected some delay due to rate limiting, took only %v", duration)
	}

	if attempt != 2 {
		t.Errorf("Expected 2 attempts (1 failure + 1 success), got %d", attempt)
	}
}
