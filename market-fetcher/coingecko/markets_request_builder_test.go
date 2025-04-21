package coingecko

import (
	"net/url"
	"strings"
	"testing"
)

func TestMarketsRequestBuilder_BuildURL(t *testing.T) {
	baseURL := "https://api.coingecko.com"

	tests := []struct {
		name          string
		configuration func(*MarketsRequestBuilder)
		checkURL      func(*testing.T, string)
	}{
		{
			name: "Default parameters",
			configuration: func(rb *MarketsRequestBuilder) {
				// Using default configuration
			},
			checkURL: func(t *testing.T, urlStr string) {
				if !strings.HasPrefix(urlStr, baseURL+"/api/v3/coins/markets") {
					t.Errorf("URL should start with %s/api/v3/coins/markets, got %s", baseURL, urlStr)
				}

				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					t.Fatalf("Failed to parse URL: %v", err)
				}

				query := parsedURL.Query()

				// Check default parameters
				if query.Get("vs_currency") != "usd" {
					t.Errorf("Expected default currency 'usd', got %s", query.Get("vs_currency"))
				}

				if query.Get("order") != "market_cap_desc" {
					t.Errorf("Expected default order 'market_cap_desc', got %s", query.Get("order"))
				}

				// API key should not be present by default
				if query.Has("x_cg_pro_api_key") || query.Has("x_cg_demo_api_key") {
					t.Error("API key parameter should not be present in default URL")
				}
			},
		},
		{
			name: "With pagination",
			configuration: func(rb *MarketsRequestBuilder) {
				rb.WithPage(2).WithPerPage(50)
			},
			checkURL: func(t *testing.T, urlStr string) {
				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					t.Fatalf("Failed to parse URL: %v", err)
				}

				query := parsedURL.Query()

				if query.Get("page") != "2" {
					t.Errorf("Expected page '2', got %s", query.Get("page"))
				}

				if query.Get("per_page") != "50" {
					t.Errorf("Expected per_page '50', got %s", query.Get("per_page"))
				}
			},
		},
		{
			name: "With Pro API key",
			configuration: func(rb *MarketsRequestBuilder) {
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
			configuration: func(rb *MarketsRequestBuilder) {
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
		{
			name: "With custom currency and order",
			configuration: func(rb *MarketsRequestBuilder) {
				rb.WithCurrency("eur").WithOrder("volume_desc")
			},
			checkURL: func(t *testing.T, urlStr string) {
				parsedURL, err := url.Parse(urlStr)
				if err != nil {
					t.Fatalf("Failed to parse URL: %v", err)
				}

				query := parsedURL.Query()

				if query.Get("vs_currency") != "eur" {
					t.Errorf("Expected currency 'eur', got %s", query.Get("vs_currency"))
				}

				if query.Get("order") != "volume_desc" {
					t.Errorf("Expected order 'volume_desc', got %s", query.Get("order"))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new builder
			rb := NewMarketRequestBuilder(baseURL)

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

func TestMarketsRequestBuilder_Build(t *testing.T) {
	baseURL := "https://api.coingecko.com"

	// Create builder with custom user agent and header
	rb := NewMarketRequestBuilder(baseURL)
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
	if !strings.HasPrefix(req.URL.String(), baseURL+"/api/v3/coins/markets") {
		t.Errorf("URL should start with %s/api/v3/coins/markets, got %s", baseURL, req.URL.String())
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
