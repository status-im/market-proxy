package coingecko

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/status-im/market-proxy/metrics"
)

// RetryOptions configures retry behavior for HTTP requests
type RetryOptions struct {
	MaxRetries        int
	BaseBackoff       time.Duration
	LogPrefix         string
	ConnectionTimeout time.Duration // Timeout for establishing connection
	RequestTimeout    time.Duration // Total request timeout including reading response
}

// DefaultRetryOptions returns default retry options
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxRetries:        3,
		BaseBackoff:       1000 * time.Millisecond,
		LogPrefix:         "HTTP",
		ConnectionTimeout: 10 * time.Second, // Default 10s connection timeout
		RequestTimeout:    30 * time.Second, // Default 30s total request timeout
	}
}

// HTTPClientWithRetries wraps an HTTP client with retry capabilities
type HTTPClientWithRetries struct {
	client *http.Client
	opts   RetryOptions
}

// NewHTTPClientWithRetries creates a new HTTP client with retry capabilities
func NewHTTPClientWithRetries(opts RetryOptions) *HTTPClientWithRetries {
	client := &http.Client{
		Timeout: opts.RequestTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: opts.ConnectionTimeout,
			}).DialContext,
		},
	}

	return &HTTPClientWithRetries{
		client: client,
		opts:   opts,
	}
}

// ExecuteRequest executes an HTTP request with retry logic
func (c *HTTPClientWithRetries) ExecuteRequest(req *http.Request, serviceName string) (*http.Response, []byte, time.Duration, error) {
	var lastErr error

	for attempt := 0; attempt < c.opts.MaxRetries; attempt++ {
		// Only log retry attempts after the first one
		if attempt > 0 {
			log.Printf("%s: Retry %d/%d after error: %v",
				c.opts.LogPrefix, attempt, c.opts.MaxRetries-1, lastErr)

			// Record retry attempt in metrics
			metrics.RecordHTTPRetry(serviceName)

			// Calculate backoff with jitter
			backoffDuration := calculateBackoffWithJitter(c.opts.BaseBackoff, attempt)
			log.Printf("%s: Waiting %.2fs before retry", c.opts.LogPrefix, backoffDuration.Seconds())
			time.Sleep(backoffDuration)
		}

		// Start time for measuring request duration
		requestStart := time.Now()

		// Execute request
		resp, err := c.client.Do(req)
		requestDuration := time.Since(requestStart)

		if err != nil {
			lastErr = fmt.Errorf("request failed after %.2fs: %v", requestDuration.Seconds(), err)
			// Record error in metrics
			metrics.RecordHTTPRequest(serviceName, "error")
			continue
		}

		// Process response
		// Extract page parameter for consistent logging
		pageContext := 0
		if page, exists := extractPageFromRequest(req); exists {
			pageContext = page
		}

		responseBody, err := processResponse(resp, pageContext, requestDuration)
		if err != nil {
			// Check if we should retry this error or give up
			if isRetryableError(resp.StatusCode) {
				lastErr = err
				resp.Body.Close()
				// Record rate limited request
				metrics.RecordHTTPRequest(serviceName, "rate_limited")
				continue
			}

			// For non-retryable errors, fail immediately
			resp.Body.Close()
			// Record general error
			metrics.RecordHTTPRequest(serviceName, "error")
			return nil, nil, requestDuration, err
		}

		// Record successful request
		metrics.RecordHTTPRequest(serviceName, "success")
		return resp, responseBody, requestDuration, nil
	}

	// If we got here, all retries failed
	return nil, nil, 0, fmt.Errorf("all %d attempts failed, last error: %v",
		c.opts.MaxRetries, lastErr)
}

// calculateBackoffWithJitter calculates backoff duration with jitter for retries
func calculateBackoffWithJitter(baseBackoff time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		return baseBackoff
	}

	multiplier := uint(1) << uint(attempt-1)
	backoff := time.Duration(float64(baseBackoff) * float64(multiplier))
	jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
	return backoff + jitter
}

// extractPageFromRequest tries to extract page parameter from request URL
func extractPageFromRequest(req *http.Request) (int, bool) {
	// Extract page from query parameters if exists
	if pageStr := req.URL.Query().Get("page"); pageStr != "" {
		var page int
		_, err := fmt.Sscanf(pageStr, "%d", &page)
		if err == nil && page > 0 {
			return page, true
		}
	}
	return 0, false
}

// processResponse reads and processes the HTTP response
func processResponse(resp *http.Response, page int, requestDuration time.Duration) ([]byte, error) {
	// Check for rate limit or other errors
	if resp.StatusCode != http.StatusOK {
		// Read error body for more details
		body, _ := io.ReadAll(resp.Body)

		// Determine if this is a rate limit issue
		if resp.StatusCode == http.StatusTooManyRequests {
			// Check for retry-after header
			retryAfter := resp.Header.Get("Retry-After")
			return nil, fmt.Errorf("rate limit exceeded (status %d), retry after %s: %s",
				resp.StatusCode, retryAfter, string(body))
		}

		return nil, fmt.Errorf("API request failed with status %d after %.2fs: %s",
			resp.StatusCode, requestDuration.Seconds(), string(body))
	}

	// Try to read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Log response size
	log.Printf("%s: Response size for page %d: %.2f KB", resp.Request.Host, page, float64(len(responseBody))/1024)

	return responseBody, nil
}

// isRetryableError determines if a given HTTP status code should trigger a retry
func isRetryableError(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || // 429 Too Many Requests
		statusCode == http.StatusInternalServerError || // 500 Internal Server Error
		statusCode == http.StatusBadGateway || // 502 Bad Gateway
		statusCode == http.StatusServiceUnavailable || // 503 Service Unavailable
		statusCode == http.StatusGatewayTimeout // 504 Gateway Timeout
}
