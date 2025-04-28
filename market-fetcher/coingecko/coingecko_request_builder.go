package coingecko

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	// Base URL for public API
	COINGECKO_PUBLIC_URL = "https://api.coingecko.com"
	// Base URL for Pro API
	COINGECKO_PRO_URL = "https://pro-api.coingecko.com"
)

// buildURL safely combines a base URL with a path
func buildURL(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	trimmedPath := strings.TrimLeft(path, "/")

	return baseURL + "/" + trimmedPath
}

// CoingeckoRequestBuilder implements the Builder pattern for CoinGecko API requests
type CoingeckoRequestBuilder struct {
	// Basic request parameters
	baseURL    string
	httpMethod string
	apiPath    string

	// Request specific parameters
	params map[string]string

	// API key information
	apiKey  string
	keyType KeyType

	// Other options
	userAgent string
	headers   map[string]string
}

// NewCoingeckoRequestBuilder creates a new base request builder for CoinGecko endpoints
func NewCoingeckoRequestBuilder(baseURL, apiPath string) *CoingeckoRequestBuilder {
	rb := &CoingeckoRequestBuilder{
		baseURL:    baseURL,
		apiPath:    apiPath,
		httpMethod: "GET",
		params:     make(map[string]string),
		headers:    make(map[string]string),
		userAgent:  "Mozilla/5.0 Market-Proxy",
	}

	// Add default headers
	rb.headers["Accept"] = "application/json"

	return rb
}

// With adds a custom parameter to the URL query
func (rb *CoingeckoRequestBuilder) With(key, value string) *CoingeckoRequestBuilder {
	rb.params[key] = value
	return rb
}

// WithApiKey sets the API key and its type
func (rb *CoingeckoRequestBuilder) WithApiKey(apiKey string, keyType KeyType) *CoingeckoRequestBuilder {
	rb.apiKey = apiKey
	rb.keyType = keyType
	return rb
}

// WithHeader adds a custom HTTP header
func (rb *CoingeckoRequestBuilder) WithHeader(name, value string) *CoingeckoRequestBuilder {
	rb.headers[name] = value
	return rb
}

// WithUserAgent sets the User-Agent header
func (rb *CoingeckoRequestBuilder) WithUserAgent(userAgent string) *CoingeckoRequestBuilder {
	rb.userAgent = userAgent
	return rb
}

// GetApiKey returns the API key and its type
func (rb *CoingeckoRequestBuilder) GetApiKey() (string, KeyType) {
	return rb.apiKey, rb.keyType
}

// BuildURL builds the complete URL for the request
func (rb *CoingeckoRequestBuilder) BuildURL() string {
	// Build the full URL using the safe path combiner
	fullPath := buildURL(rb.baseURL, rb.apiPath)

	// Create query parameters
	query := url.Values{}

	// Add all parameters
	for key, value := range rb.params {
		query.Add(key, value)
	}

	// Add API key if available
	if rb.apiKey != "" {
		switch rb.keyType {
		case ProKey:
			query.Add("x_cg_pro_api_key", rb.apiKey)
		case DemoKey:
			query.Add("x_cg_demo_api_key", rb.apiKey)
		}
	}

	// Combine URL and query parameters
	finalURL := fullPath
	queryString := query.Encode()
	if queryString != "" {
		finalURL = fmt.Sprintf("%s?%s", finalURL, queryString)
	}

	return finalURL
}

// Build creates an http.Request object
func (rb *CoingeckoRequestBuilder) Build() (*http.Request, error) {
	// Build the URL
	finalURL := rb.BuildURL()

	// Create the request
	req, err := http.NewRequest(rb.httpMethod, finalURL, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent
	req.Header.Set("User-Agent", rb.userAgent)

	// Add all headers
	for key, value := range rb.headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
