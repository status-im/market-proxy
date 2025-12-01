package coingecko_common

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
	baseURL    string
	httpMethod string
	apiPath    string
	params     map[string]string
	apiKey     string
	keyType    KeyType
	userAgent  string
	headers    map[string]string
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

	rb.headers["Accept"] = "application/json"

	return rb
}

// With adds a custom parameter to the URL query
func (rb *CoingeckoRequestBuilder) With(key, value string) *CoingeckoRequestBuilder {
	rb.params[key] = value
	return rb
}

// WithCurrency adds vs_currency parameter
func (rb *CoingeckoRequestBuilder) WithCurrency(currency string) *CoingeckoRequestBuilder {
	if currency != "" {
		rb.params["vs_currency"] = currency
	}
	return rb
}

// WithApiKey sets the API key and its type
func (rb *CoingeckoRequestBuilder) WithApiKey(apiKey string, keyType KeyType) *CoingeckoRequestBuilder {
	if apiKey != "" {
		rb.apiKey = apiKey
		rb.keyType = keyType
	}
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
	fullPath := buildURL(rb.baseURL, rb.apiPath)

	query := url.Values{}

	for key, value := range rb.params {
		query.Add(key, value)
	}

	if rb.apiKey != "" {
		switch rb.keyType {
		case ProKey:
			query.Add("x_cg_pro_api_key", rb.apiKey)
		case DemoKey:
			query.Add("x_cg_demo_api_key", rb.apiKey)
		}
	}

	finalURL := fullPath
	queryString := query.Encode()
	if queryString != "" {
		finalURL = fmt.Sprintf("%s?%s", finalURL, queryString)
	}

	return finalURL
}

// Build creates an http.Request object
func (rb *CoingeckoRequestBuilder) Build() (*http.Request, error) {
	return rb.BuildWithURL(rb.BuildURL())
}

func (rb *CoingeckoRequestBuilder) BuildWithURL(finalURL string) (*http.Request, error) {
	req, err := http.NewRequest(rb.httpMethod, finalURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", rb.userAgent)

	for key, value := range rb.headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
