package coingecko_common

import (
	"net/http"

	hc "github.com/status-im/proxy-common/httpclient"
	"golang.org/x/time/rate"
)

// Aliases to proxy-common/httpclient (identical contract except rate limiting hook).
type (
	HTTPClientWithRetries = hc.HTTPClientWithRetries
	RetryOptions          = hc.RetryOptions
	IHttpStatusHandler    = hc.IHttpStatusHandler
)

// DefaultRetryOptions returns default retry options.
func DefaultRetryOptions() RetryOptions {
	return hc.DefaultRetryOptions()
}

// NewHTTPClientWithRetries creates an HTTP client with retries; IRateLimiterManager
// is adapted to the callback used by proxy-common/httpclient.
func NewHTTPClientWithRetries(opts RetryOptions, handler IHttpStatusHandler, lm IRateLimiterManager) *HTTPClientWithRetries {
	var rl func(*http.Request) *rate.Limiter
	if lm != nil {
		rl = func(req *http.Request) *rate.Limiter { return lm.GetLimiterForURL(req.URL) }
	}
	return hc.NewHTTPClientWithRetries(opts, handler, rl)
}
