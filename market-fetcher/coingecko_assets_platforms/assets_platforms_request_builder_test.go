package coingecko_assets_platforms

import (
	"testing"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/stretchr/testify/assert"
)

func TestNewAssetsPlatformsRequestBuilder(t *testing.T) {
	baseURL := "https://api.coingecko.com"
	builder := NewAssetsPlatformsRequestBuilder(baseURL)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.builder)
}

func TestAssetsPlatformsRequestBuilder_WithFilter(t *testing.T) {
	tests := []struct {
		name          string
		filter        string
		expectedInURL bool
		expectedParam string
	}{
		{
			name:          "With valid filter",
			filter:        "ethereum",
			expectedInURL: true,
			expectedParam: "filter=ethereum",
		},
		{
			name:          "With empty filter",
			filter:        "",
			expectedInURL: false,
			expectedParam: "",
		},
		{
			name:          "With spaces in filter",
			filter:        "ethereum mainnet",
			expectedInURL: true,
			expectedParam: "filter=ethereum+mainnet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL := "https://api.coingecko.com"
			builder := NewAssetsPlatformsRequestBuilder(baseURL)

			result := builder.WithFilter(tt.filter)

			// Test method chaining
			assert.Equal(t, builder, result)

			// Build the request to check URL
			req, err := builder.builder.Build()
			assert.NoError(t, err)

			if tt.expectedInURL {
				assert.Contains(t, req.URL.RawQuery, tt.expectedParam)
			} else {
				assert.NotContains(t, req.URL.RawQuery, "filter=")
			}
		})
	}
}

func TestAssetsPlatformsRequestBuilder_Build(t *testing.T) {
	tests := []struct {
		name            string
		baseURL         string
		filter          string
		apiKey          string
		keyType         cg.KeyType
		expectedPath    string
		expectedQueries []string
	}{
		{
			name:            "Basic request without parameters",
			baseURL:         "https://api.coingecko.com",
			filter:          "",
			apiKey:          "",
			keyType:         cg.NoKey,
			expectedPath:    "/api/v3/asset_platforms",
			expectedQueries: []string{},
		},
		{
			name:            "Request with filter",
			baseURL:         "https://api.coingecko.com",
			filter:          "ethereum",
			apiKey:          "",
			keyType:         cg.NoKey,
			expectedPath:    "/api/v3/asset_platforms",
			expectedQueries: []string{"filter=ethereum"},
		},
		{
			name:            "Request with Pro API key",
			baseURL:         "https://pro-api.coingecko.com",
			filter:          "",
			apiKey:          "test-pro-key",
			keyType:         cg.ProKey,
			expectedPath:    "/api/v3/asset_platforms",
			expectedQueries: []string{"x_cg_pro_api_key=test-pro-key"},
		},
		{
			name:            "Request with Demo API key",
			baseURL:         "https://api.coingecko.com",
			filter:          "",
			apiKey:          "test-demo-key",
			keyType:         cg.DemoKey,
			expectedPath:    "/api/v3/asset_platforms",
			expectedQueries: []string{"x_cg_demo_api_key=test-demo-key"},
		},
		{
			name:            "Request with filter and API key",
			baseURL:         "https://pro-api.coingecko.com",
			filter:          "polygon",
			apiKey:          "test-key",
			keyType:         cg.ProKey,
			expectedPath:    "/api/v3/asset_platforms",
			expectedQueries: []string{"filter=polygon", "x_cg_pro_api_key=test-key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewAssetsPlatformsRequestBuilder(tt.baseURL)

			if tt.filter != "" {
				builder.WithFilter(tt.filter)
			}

			if tt.apiKey != "" {
				builder.builder.WithApiKey(tt.apiKey, tt.keyType)
			}

			req, err := builder.builder.Build()

			assert.NoError(t, err)
			assert.NotNil(t, req)
			assert.Equal(t, "GET", req.Method)
			assert.Contains(t, req.URL.Path, tt.expectedPath)

			// Check all expected query parameters are present
			for _, expectedQuery := range tt.expectedQueries {
				assert.Contains(t, req.URL.RawQuery, expectedQuery)
			}

			// Check common headers
			assert.Equal(t, "application/json", req.Header.Get("Accept"))
			assert.NotEmpty(t, req.Header.Get("User-Agent"))
		})
	}
}

func TestAssetsPlatformsRequestBuilder_URLConstruction(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		expectedURL string
	}{
		{
			name:        "Standard public URL",
			baseURL:     "https://api.coingecko.com",
			expectedURL: "https://api.coingecko.com/api/v3/asset_platforms",
		},
		{
			name:        "Pro API URL",
			baseURL:     "https://pro-api.coingecko.com",
			expectedURL: "https://pro-api.coingecko.com/api/v3/asset_platforms",
		},
		{
			name:        "URL with trailing slash",
			baseURL:     "https://api.coingecko.com/",
			expectedURL: "https://api.coingecko.com/api/v3/asset_platforms",
		},
		{
			name:        "Custom base URL",
			baseURL:     "https://custom-api.example.com",
			expectedURL: "https://custom-api.example.com/api/v3/asset_platforms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewAssetsPlatformsRequestBuilder(tt.baseURL)
			req, err := builder.builder.Build()

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedURL, req.URL.String())
		})
	}
}

func TestAssetsPlatformsRequestBuilder_MethodChaining(t *testing.T) {
	baseURL := "https://api.coingecko.com"
	builder := NewAssetsPlatformsRequestBuilder(baseURL)

	// Test that all methods return the same builder instance for chaining
	result := builder.WithFilter("ethereum")
	assert.Equal(t, builder, result)

	// Test multiple chained calls
	finalBuilder := NewAssetsPlatformsRequestBuilder(baseURL).
		WithFilter("polygon")

	req, err := finalBuilder.builder.Build()
	assert.NoError(t, err)
	assert.Contains(t, req.URL.RawQuery, "filter=polygon")
}
