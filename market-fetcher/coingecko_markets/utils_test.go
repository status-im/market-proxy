package coingecko_markets

import (
	"testing"

	"github.com/status-im/market-proxy/interfaces"

	"github.com/stretchr/testify/assert"
)

func TestGetCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		tokenID  string
		expected string
	}{
		{
			name:     "Valid token ID",
			tokenID:  "bitcoin",
			expected: "markets:bitcoin",
		},
		{
			name:     "Empty token ID",
			tokenID:  "",
			expected: "markets:",
		},
		{
			name:     "Token ID with special characters",
			tokenID:  "ethereum-classic",
			expected: "markets:ethereum-classic",
		},
		{
			name:     "Numeric token ID",
			tokenID:  "1inch",
			expected: "markets:1inch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCacheKey(tt.tokenID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTokenIDFromKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "Valid cache key",
			key:      "markets:bitcoin",
			expected: "bitcoin",
		},
		{
			name:     "Valid cache key with special characters",
			key:      "markets:ethereum-classic",
			expected: "ethereum-classic",
		},
		{
			name:     "Valid cache key with empty token ID",
			key:      "markets:",
			expected: "",
		},
		{
			name:     "Invalid key without prefix",
			key:      "bitcoin",
			expected: "",
		},
		{
			name:     "Empty key",
			key:      "",
			expected: "",
		},
		{
			name:     "Key with wrong prefix",
			key:      "prices:bitcoin",
			expected: "",
		},
		{
			name:     "Partial prefix match",
			key:      "market:bitcoin",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTokenIDFromKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateCacheKeys(t *testing.T) {
	tests := []struct {
		name     string
		params   interfaces.MarketsParams
		expected []string
	}{
		{
			name: "Valid params with multiple IDs",
			params: interfaces.MarketsParams{
				IDs: []string{"bitcoin", "ethereum", "litecoin"},
			},
			expected: []string{"markets:bitcoin", "markets:ethereum", "markets:litecoin"},
		},
		{
			name: "Valid params with single ID",
			params: interfaces.MarketsParams{
				IDs: []string{"bitcoin"},
			},
			expected: []string{"markets:bitcoin"},
		},
		{
			name: "Empty IDs slice",
			params: interfaces.MarketsParams{
				IDs: []string{},
			},
			expected: []string{},
		},
		{
			name: "Nil IDs slice",
			params: interfaces.MarketsParams{
				IDs: nil,
			},
			expected: []string{},
		},
		{
			name: "IDs with empty string",
			params: interfaces.MarketsParams{
				IDs: []string{"bitcoin", "", "ethereum"},
			},
			expected: []string{"markets:bitcoin", "markets:", "markets:ethereum"},
		},
		{
			name: "IDs with special characters",
			params: interfaces.MarketsParams{
				IDs: []string{"ethereum-classic", "1inch", "polygon-ecosystem-token"},
			},
			expected: []string{"markets:ethereum-classic", "markets:1inch", "markets:polygon-ecosystem-token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createCacheKeys(tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTokensFromKeys(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		expected []string
	}{
		{
			name:     "Valid cache keys",
			keys:     []string{"markets:bitcoin", "markets:ethereum", "markets:litecoin"},
			expected: []string{"bitcoin", "ethereum", "litecoin"},
		},
		{
			name:     "Single cache key",
			keys:     []string{"markets:bitcoin"},
			expected: []string{"bitcoin"},
		},
		{
			name:     "Empty keys slice",
			keys:     []string{},
			expected: []string{},
		},
		{
			name:     "Nil keys slice",
			keys:     nil,
			expected: []string{},
		},
		{
			name:     "Mixed valid and invalid keys",
			keys:     []string{"markets:bitcoin", "invalid:ethereum", "markets:litecoin"},
			expected: []string{"bitcoin", "litecoin"},
		},
		{
			name:     "All invalid keys",
			keys:     []string{"invalid:bitcoin", "wrong:ethereum", "bad:litecoin"},
			expected: []string{},
		},
		{
			name:     "Duplicate token IDs",
			keys:     []string{"markets:bitcoin", "markets:ethereum", "markets:bitcoin"},
			expected: []string{"bitcoin", "ethereum"},
		},
		{
			name:     "Keys with empty token IDs",
			keys:     []string{"markets:", "markets:bitcoin", "markets:"},
			expected: []string{"bitcoin"},
		},
		{
			name:     "Keys with special characters",
			keys:     []string{"markets:ethereum-classic", "markets:1inch", "markets:polygon-ecosystem-token"},
			expected: []string{"ethereum-classic", "1inch", "polygon-ecosystem-token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTokensFromKeys(tt.keys)

			// Since the function returns tokens in map iteration order (not guaranteed),
			// we need to check that all expected tokens are present
			assert.Len(t, result, len(tt.expected))
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected)
			}
		})
	}
}
