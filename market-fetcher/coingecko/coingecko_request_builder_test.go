package coingecko

import (
	"net/url"
	"strings"
	"testing"
)

func TestCoingeckoRequestBuilder_BuildURL(t *testing.T) {
	baseURL := "https://api.coingecko.com"
	apiPath := "/api/v3/test-endpoint"

	tests := []struct {
		name          string
		configuration func(*CoingeckoRequestBuilder)
		checkURL      func(*testing.T, string)
	}{
		{
			name: "Default parameters",
			configuration: func(rb *CoingeckoRequestBuilder) {
				// Using default configuration
			},
			checkURL: func(t *testing.T, urlStr string) {
				if !strings.HasPrefix(urlStr, baseURL+apiPath) {
					t.Errorf("URL should start with %s%s, got %s", baseURL, apiPath, urlStr)
				}

				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					t.Fatalf("Failed to parse URL: %v", err)
				}

				// API key should not be present by default
				query := parsedURL.Query()
				if query.Has("x_cg_pro_api_key") || query.Has("x_cg_demo_api_key") {
					t.Error("API key parameter should not be present in default URL")
				}
			},
		},
		{
			name: "With parameters",
			configuration: func(rb *CoingeckoRequestBuilder) {
				rb.With("param1", "value1").With("param2", "value2")
			},
			checkURL: func(t *testing.T, urlStr string) {
				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					t.Fatalf("Failed to parse URL: %v", err)
				}

				query := parsedURL.Query()

				if query.Get("param1") != "value1" {
					t.Errorf("Expected param1 'value1', got %s", query.Get("param1"))
				}

				if query.Get("param2") != "value2" {
					t.Errorf("Expected param2 'value2', got %s", query.Get("param2"))
				}
			},
		},
		{
			name: "With Pro API key",
			configuration: func(rb *CoingeckoRequestBuilder) {
				rb.WithApiKey("test-pro-key", ProKey)
			},
			checkURL: func(t *testing.T, urlStr string) {
				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					t.Fatalf("Failed to parse URL: %v", err)
				}

				query := parsedURL.Query()

				if query.Get("x_cg_pro_api_key") != "test-pro-key" {
					t.Errorf("Expected Pro API key 'test-pro-key', got %s", query.Get("x_cg_pro_api_key"))
				}

				if query.Has("x_cg_demo_api_key") {
					t.Error("Demo API key parameter should not be present")
				}
			},
		},
		{
			name: "With Demo API key",
			configuration: func(rb *CoingeckoRequestBuilder) {
				rb.WithApiKey("test-demo-key", DemoKey)
			},
			checkURL: func(t *testing.T, urlStr string) {
				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					t.Fatalf("Failed to parse URL: %v", err)
				}

				query := parsedURL.Query()

				if query.Get("x_cg_demo_api_key") != "test-demo-key" {
					t.Errorf("Expected Demo API key 'test-demo-key', got %s", query.Get("x_cg_demo_api_key"))
				}

				if query.Has("x_cg_pro_api_key") {
					t.Error("Pro API key parameter should not be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new builder
			rb := NewCoingeckoRequestBuilder(baseURL, apiPath)

			// Apply configuration
			if tt.configuration != nil {
				tt.configuration(rb)
			}

			// Build URL
			url := rb.BuildURL()

			// Check URL
			if tt.checkURL != nil {
				tt.checkURL(t, url)
			}
		})
	}
}

func TestCoingeckoRequestBuilder_Build(t *testing.T) {
	baseURL := "https://api.coingecko.com"
	apiPath := "/api/v3/test-endpoint"

	// Create builder with custom user agent and header
	rb := NewCoingeckoRequestBuilder(baseURL, apiPath)
	rb.WithUserAgent("TestAgent/1.0")
	rb.WithHeader("X-Test-Header", "test-value")

	// Build request
	req, err := rb.Build()
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	// Check method
	if req.Method != "GET" {
		t.Errorf("Expected method 'GET', got %s", req.Method)
	}

	// Check URL
	if !strings.HasPrefix(req.URL.String(), baseURL+apiPath) {
		t.Errorf("URL should start with %s%s, got %s", baseURL, apiPath, req.URL.String())
	}

	// Check headers
	if req.Header.Get("User-Agent") != "TestAgent/1.0" {
		t.Errorf("Expected User-Agent 'TestAgent/1.0', got %s", req.Header.Get("User-Agent"))
	}

	if req.Header.Get("X-Test-Header") != "test-value" {
		t.Errorf("Expected custom header value 'test-value', got %s", req.Header.Get("X-Test-Header"))
	}

	if req.Header.Get("Accept") != "application/json" {
		t.Errorf("Expected Accept header 'application/json', got %s", req.Header.Get("Accept"))
	}
}

// TestGetApiKey verifies the GetApiKey method
func TestCoingeckoRequestBuilder_GetApiKey(t *testing.T) {
	rb := NewCoingeckoRequestBuilder("https://api.coingecko.com", "/api/v3/test-endpoint")

	// Default should be empty key and NoKey type
	key, keyType := rb.GetApiKey()
	if key != "" || keyType != NoKey {
		t.Errorf("Expected empty key and NoKey type, got %s and %v", key, keyType)
	}

	// Set API key and check if it's correctly returned
	rb.WithApiKey("test-key", ProKey)
	key, keyType = rb.GetApiKey()
	if key != "test-key" || keyType != ProKey {
		t.Errorf("Expected 'test-key' and ProKey type, got %s and %v", key, keyType)
	}
}
