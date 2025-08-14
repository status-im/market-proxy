package coingecko_common

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

// IHttpStatusHandler is an interface for handling HTTP request statuses
type IHttpStatusHandler interface {
	// OnRequest handles a request with its status result
	OnRequest(status string)
	// OnRetry handles retry events
	OnRetry()
}

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

// HTTPClientWithRetries wraps an HTTP Client with retry capabilities
type HTTPClientWithRetries struct {
	Client         *http.Client
	Opts           RetryOptions
	StatusHandler  IHttpStatusHandler
	LimiterManager IRateLimiterManager
}

// NewHTTPClientWithRetries creates a new HTTP Client with retry capabilities
func NewHTTPClientWithRetries(opts RetryOptions, handler IHttpStatusHandler, limiterManager IRateLimiterManager) *HTTPClientWithRetries {
	client := &http.Client{
		Timeout: opts.RequestTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: opts.ConnectionTimeout,
			}).DialContext,
		},
	}

	return &HTTPClientWithRetries{
		Client:         client,
		Opts:           opts,
		StatusHandler:  handler,
		LimiterManager: limiterManager,
	}
}

// SetStatusHandler sets the status handler for this Client
func (c *HTTPClientWithRetries) SetStatusHandler(handler IHttpStatusHandler) {
	c.StatusHandler = handler
}

// ExecuteRequest executes an HTTP request with retry logic
func (c *HTTPClientWithRetries) ExecuteRequest(req *http.Request) (*http.Response, []byte, time.Duration, error) {
	var lastErr error

	for attempt := 0; attempt < c.Opts.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("%s: Retry %d/%d after error: %v",
				c.Opts.LogPrefix, attempt, c.Opts.MaxRetries-1, lastErr)

			if c.StatusHandler != nil {
				c.StatusHandler.OnRetry()
			}

			backoffDuration := calculateBackoffWithJitter(c.Opts.BaseBackoff, attempt)
			log.Printf("%s: Waiting %.2fs before retry", c.Opts.LogPrefix, backoffDuration.Seconds())
			time.Sleep(backoffDuration)
		}

		requestStart := time.Now()

		// Rate limit per API key before executing the request
		if c.LimiterManager != nil {
			limiter := c.LimiterManager.GetLimiterForURL(req.URL)
			if limiter != nil {
				if err := limiter.Wait(req.Context()); err != nil {
					lastErr = fmt.Errorf("rate limiter wait failed: %w", err)
					if c.StatusHandler != nil {
						c.StatusHandler.OnRequest("error")
					}
					break
				}
			}
		}

		// Execute request
		resp, err := c.Client.Do(req)
		requestDuration := time.Since(requestStart)

		if err != nil {
			lastErr = fmt.Errorf("request failed after %.2fs: %v", requestDuration.Seconds(), err)
			if c.StatusHandler != nil {
				c.StatusHandler.OnRequest("error")
			}
			continue
		}

		pageContext := 0
		if page, exists := extractPageFromRequest(req); exists {
			pageContext = page
		}

		responseBody, err := processResponse(resp, pageContext, requestDuration)
		if err != nil {
			if isRetryableError(resp.StatusCode) {
				lastErr = err
				resp.Body.Close()
				if c.StatusHandler != nil {
					c.StatusHandler.OnRequest("rate_limited")
				}
				continue
			}

			resp.Body.Close()
			if c.StatusHandler != nil {
				c.StatusHandler.OnRequest("error")
			}
			return nil, nil, requestDuration, err
		}

		if c.StatusHandler != nil {
			c.StatusHandler.OnRequest("success")
		}
		return resp, responseBody, requestDuration, nil
	}

	return nil, nil, 0, fmt.Errorf("all %d attempts failed, last error: %v",
		c.Opts.MaxRetries, lastErr)
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
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			return nil, fmt.Errorf("rate limit exceeded (status %d), retry after %s: %s",
				resp.StatusCode, retryAfter, string(body))
		}

		return nil, fmt.Errorf("API request failed with status %d after %.2fs: %s",
			resp.StatusCode, requestDuration.Seconds(), string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	return responseBody, nil
}

// isRetryableError determines if a given HTTP status code should trigger a retry
func isRetryableError(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}
