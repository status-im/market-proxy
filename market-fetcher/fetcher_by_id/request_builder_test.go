package fetcher_by_id

import (
	"testing"

	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
)

func TestRequestBuilder_BuildSingleURL(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		endpointPath string
		params       map[string]interface{}
		id           string
		wantContains []string
		wantErr      bool
	}{
		{
			name:         "basic single URL",
			baseURL:      "https://api.coingecko.com",
			endpointPath: "/api/v3/coins/{{id}}",
			id:           "bitcoin",
			wantContains: []string{"https://api.coingecko.com/api/v3/coins/bitcoin"},
			wantErr:      false,
		},
		{
			name:         "URL with query params",
			baseURL:      "https://api.coingecko.com",
			endpointPath: "/api/v3/coins/{{id}}",
			params: map[string]interface{}{
				"localization": false,
				"tickers":      false,
			},
			id:           "ethereum",
			wantContains: []string{"ethereum", "localization=false", "tickers=false"},
			wantErr:      false,
		},
		{
			name:         "missing placeholder",
			baseURL:      "https://api.coingecko.com",
			endpointPath: "/api/v3/coins/bitcoin",
			id:           "bitcoin",
			wantErr:      true,
		},
		{
			name:         "URL with special characters in ID",
			baseURL:      "https://api.coingecko.com",
			endpointPath: "/api/v3/coins/{{id}}",
			id:           "wrapped-bitcoin",
			wantContains: []string{"wrapped-bitcoin"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.FetcherByIdConfig{
				EndpointPath:   tt.endpointPath,
				ParamsOverride: tt.params,
			}
			builder := NewRequestBuilder(tt.baseURL, cfg)

			url, err := builder.BuildSingleURL(tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				for _, substr := range tt.wantContains {
					assert.Contains(t, url, substr)
				}
			}
		})
	}
}

func TestRequestBuilder_BuildBatchURL(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		endpointPath string
		params       map[string]interface{}
		ids          []string
		wantContains []string
		wantErr      bool
	}{
		{
			name:         "basic batch URL",
			baseURL:      "https://api.coingecko.com",
			endpointPath: "/api/v3/simple/price?ids={{ids_list}}",
			ids:          []string{"bitcoin", "ethereum"},
			wantContains: []string{"bitcoin", "ethereum"},
			wantErr:      false,
		},
		{
			name:         "batch URL with query params",
			baseURL:      "https://api.coingecko.com",
			endpointPath: "/api/v3/simple/price?ids={{ids_list}}",
			params: map[string]interface{}{
				"vs_currencies": "usd",
			},
			ids:          []string{"bitcoin"},
			wantContains: []string{"bitcoin", "vs_currencies=usd"},
			wantErr:      false,
		},
		{
			name:         "missing placeholder",
			baseURL:      "https://api.coingecko.com",
			endpointPath: "/api/v3/simple/price",
			ids:          []string{"bitcoin"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.FetcherByIdConfig{
				EndpointPath:   tt.endpointPath,
				ParamsOverride: tt.params,
			}
			builder := NewRequestBuilder(tt.baseURL, cfg)

			url, err := builder.BuildBatchURL(tt.ids)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				for _, substr := range tt.wantContains {
					assert.Contains(t, url, substr)
				}
			}
		})
	}
}

func TestRequestBuilder_BuildSingleRequest(t *testing.T) {
	cfg := &config.FetcherByIdConfig{
		EndpointPath: "/api/v3/coins/{{id}}",
		ParamsOverride: map[string]interface{}{
			"localization": false,
		},
	}

	builder := NewRequestBuilder("https://api.coingecko.com", cfg)
	req, err := builder.BuildSingleRequest("bitcoin")

	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, "GET", req.Method)
	assert.Contains(t, req.URL.String(), "bitcoin")
	assert.Contains(t, req.URL.String(), "localization=false")
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
	assert.Contains(t, req.Header.Get("User-Agent"), "Market-Proxy")
}

func TestRequestBuilder_BuildBatchRequest(t *testing.T) {
	cfg := &config.FetcherByIdConfig{
		EndpointPath: "/api/v3/simple/price?ids={{ids_list}}",
		ParamsOverride: map[string]interface{}{
			"vs_currencies": "usd",
		},
	}

	builder := NewRequestBuilder("https://api.coingecko.com", cfg)
	req, err := builder.BuildBatchRequest([]string{"bitcoin", "ethereum"})

	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.Equal(t, "GET", req.Method)
	assert.Contains(t, req.URL.String(), "bitcoin")
	assert.Contains(t, req.URL.String(), "ethereum")
	assert.Contains(t, req.URL.String(), "vs_currencies=usd")
}

func TestRequestBuilder_TrailingSlash(t *testing.T) {
	// Base URL with trailing slash should be handled correctly
	cfg := &config.FetcherByIdConfig{
		EndpointPath: "/api/v3/coins/{{id}}",
	}

	builder1 := NewRequestBuilder("https://api.coingecko.com/", cfg)
	url1, err := builder1.BuildSingleURL("bitcoin")
	assert.NoError(t, err)
	assert.Contains(t, url1, "https://api.coingecko.com/api/v3/coins/bitcoin")

	builder2 := NewRequestBuilder("https://api.coingecko.com", cfg)
	url2, err := builder2.BuildSingleURL("bitcoin")
	assert.NoError(t, err)

	// Both should produce the same URL (no double slashes)
	assert.Equal(t, url1, url2)
}
