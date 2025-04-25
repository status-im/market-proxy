package coingecko

import (
	"net/url"
	"strings"
	"testing"
)

func TestMarketsRequestBuilder_SpecificBehavior(t *testing.T) {
	baseURL := "https://api.coingecko.com"

	tests := []struct {
		name          string
		configuration func(*MarketsRequestBuilder)
		checkURL      func(*testing.T, string)
	}{
		{
			name: "Default market parameters",
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

				// Check default parameters specific to markets endpoint
				if query.Get("vs_currency") != "usd" {
					t.Errorf("Expected default currency 'usd', got %s", query.Get("vs_currency"))
				}

				if query.Get("order") != "market_cap_desc" {
					t.Errorf("Expected default order 'market_cap_desc', got %s", query.Get("order"))
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

func TestMarketsRequestBuilder_EndpointPath(t *testing.T) {
	baseURL := "https://api.coingecko.com"

	// Create builder
	rb := NewMarketRequestBuilder(baseURL)

	// Build request
	req, err := rb.Build()
	if err != nil {
		t.Fatalf("Failed to build request: %v", err)
	}

	// Check the correct API endpoint is used
	if !strings.HasPrefix(req.URL.String(), baseURL+"/api/v3/coins/markets") {
		t.Errorf("URL should start with %s/api/v3/coins/markets, got %s", baseURL, req.URL.String())
	}
}
