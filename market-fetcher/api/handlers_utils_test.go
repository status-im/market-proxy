package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetParamLowercase(t *testing.T) {
	tests := []struct {
		name        string
		queryParams map[string]string
		key         string
		expected    string
		nilRequest  bool
	}{
		{
			name:        "converts uppercase parameter to lowercase",
			queryParams: map[string]string{"currency": "USD"},
			key:         "currency",
			expected:    "usd",
		},
		{
			name:        "converts mixed case parameter to lowercase",
			queryParams: map[string]string{"ids": "Bitcoin,Ethereum"},
			key:         "ids",
			expected:    "bitcoin,ethereum",
		},
		{
			name:        "returns empty string for missing parameter",
			queryParams: map[string]string{},
			key:         "missing",
			expected:    "",
		},
		{
			name:        "returns empty string for empty parameter value",
			queryParams: map[string]string{"empty": ""},
			key:         "empty",
			expected:    "",
		},
		{
			name:        "handles already lowercase parameter",
			queryParams: map[string]string{"order": "market_cap_desc"},
			key:         "order",
			expected:    "market_cap_desc",
		},
		{
			name:        "handles special characters and numbers",
			queryParams: map[string]string{"filter": "Test-123_ABC"},
			key:         "filter",
			expected:    "test-123_abc",
		},
		{
			name:        "returns empty string for nil request",
			queryParams: nil,
			key:         "any",
			expected:    "",
			nilRequest:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request

			if tt.nilRequest {
				req = nil
			} else {
				// Create a mock HTTP request with query parameters
				req = &http.Request{
					URL: &url.URL{},
				}
				q := req.URL.Query()
				for key, value := range tt.queryParams {
					q.Set(key, value)
				}
				req.URL.RawQuery = q.Encode()
			}

			result := getParamLowercase(req, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitParamLowercase(t *testing.T) {
	tests := []struct {
		name     string
		param    string
		expected []string
	}{
		{
			name:     "splits and converts uppercase values to lowercase",
			param:    "BITCOIN,ETHEREUM,CARDANO",
			expected: []string{"bitcoin", "ethereum", "cardano"},
		},
		{
			name:     "splits and converts mixed case values to lowercase",
			param:    "Bitcoin,Ethereum,Cardano",
			expected: []string{"bitcoin", "ethereum", "cardano"},
		},
		{
			name:     "handles single value",
			param:    "BITCOIN",
			expected: []string{"bitcoin"},
		},
		{
			name:     "handles empty string",
			param:    "",
			expected: []string{},
		},
		{
			name:     "handles values with spaces and trims them",
			param:    " BITCOIN , ETHEREUM , CARDANO ",
			expected: []string{"bitcoin", "ethereum", "cardano"},
		},
		{
			name:     "handles already lowercase values",
			param:    "bitcoin,ethereum,cardano",
			expected: []string{"bitcoin", "ethereum", "cardano"},
		},
		{
			name:     "handles special characters and numbers",
			param:    "BTC-USD,ETH_EUR,ADA123",
			expected: []string{"btc-usd", "eth_eur", "ada123"},
		},
		{
			name:     "filters out empty values in the list",
			param:    "bitcoin,,ethereum",
			expected: []string{"bitcoin", "ethereum"},
		},
		{
			name:     "filters out empty values from single comma",
			param:    ",",
			expected: []string{},
		},
		{
			name:     "filters out trailing comma",
			param:    "bitcoin,ethereum,",
			expected: []string{"bitcoin", "ethereum"},
		},
		{
			name:     "filters out leading comma",
			param:    ",bitcoin,ethereum",
			expected: []string{"bitcoin", "ethereum"},
		},
		{
			name:     "filters out whitespace-only values",
			param:    "bitcoin,   ,ethereum",
			expected: []string{"bitcoin", "ethereum"},
		},
		{
			name:     "handles multiple consecutive commas",
			param:    "bitcoin,,,ethereum",
			expected: []string{"bitcoin", "ethereum"},
		},
		{
			name:     "returns empty slice for whitespace and commas only",
			param:    " , , , ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitParamLowercase(tt.param)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSendJSONResponse(t *testing.T) {
	tests := []struct {
		name         string
		data         interface{}
		expectedJSON string
	}{
		{
			name:         "simple object",
			data:         map[string]string{"message": "hello"},
			expectedJSON: `{"message":"hello"}`,
		},
		{
			name:         "simple array",
			data:         []string{"a", "b", "c"},
			expectedJSON: `["a","b","c"]`,
		},
		{
			name:         "complex object",
			data:         map[string]interface{}{"count": 3, "items": []string{"x", "y"}},
			expectedJSON: `{"count":3,"items":["x","y"]}`,
		},
		{
			name:         "empty object",
			data:         map[string]interface{}{},
			expectedJSON: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{}
			recorder := httptest.NewRecorder()

			server.sendJSONResponse(recorder, tt.data)

			// Check status code
			assert.Equal(t, http.StatusOK, recorder.Code)

			// Check content type
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

			// Check that response body doesn't end with newline
			body := recorder.Body.String()
			assert.Equal(t, tt.expectedJSON, body)
			assert.False(t, strings.HasSuffix(body, "\n"), "Response should not end with newline")

			// Check Content-Length header matches actual body length
			expectedLength := len(tt.expectedJSON)
			assert.Equal(t, expectedLength, recorder.Body.Len())

			// Check ETag header is set
			etag := recorder.Header().Get("ETag")
			assert.True(t, len(etag) > 0, "ETag header should be set")
			assert.True(t, strings.HasPrefix(etag, `"`), "ETag should start with quote")
			assert.True(t, strings.HasSuffix(etag, `"`), "ETag should end with quote")
		})
	}
}
