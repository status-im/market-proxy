package coingecko_tokens

import (
	"reflect"
	"testing"
)

func TestFilterTokensByPlatform(t *testing.T) {
	tests := []struct {
		name               string
		tokens             []Token
		supportedPlatforms []string
		want               []Token
	}{
		{
			name: "filter tokens with multiple platforms",
			tokens: []Token{
				{
					ID:     "bitcoin",
					Symbol: "btc",
					Name:   "Bitcoin",
					Platforms: map[string]string{
						"ethereum":    "0xbtc",
						"polygon-pos": "0xbtc-poly",
						"solana":      "solbtc",
					},
				},
				{
					ID:     "ethereum",
					Symbol: "eth",
					Name:   "Ethereum",
					Platforms: map[string]string{
						"binance-smart-chain": "0xeth-bsc",
						"polygon-pos":         "0xeth-poly",
					},
				},
				{
					ID:     "solana",
					Symbol: "sol",
					Name:   "Solana",
					Platforms: map[string]string{
						"solana": "sol",
					},
				},
			},
			supportedPlatforms: []string{"ethereum", "polygon-pos"},
			want: []Token{
				{
					ID:     "bitcoin",
					Symbol: "btc",
					Name:   "Bitcoin",
					Platforms: map[string]string{
						"ethereum":    "0xbtc",
						"polygon-pos": "0xbtc-poly",
					},
				},
				{
					ID:     "ethereum",
					Symbol: "eth",
					Name:   "Ethereum",
					Platforms: map[string]string{
						"polygon-pos": "0xeth-poly",
					},
				},
			},
		},
		{
			name: "no supported platforms",
			tokens: []Token{
				{
					ID:     "bitcoin",
					Symbol: "btc",
					Name:   "Bitcoin",
					Platforms: map[string]string{
						"ethereum":    "0xbtc",
						"polygon-pos": "0xbtc-poly",
					},
				},
			},
			supportedPlatforms: []string{},
			want:               []Token{},
		},
		{
			name:               "empty tokens list",
			tokens:             []Token{},
			supportedPlatforms: []string{"ethereum", "polygon-pos"},
			want:               []Token{},
		},
		{
			name: "no matching platforms",
			tokens: []Token{
				{
					ID:     "bitcoin",
					Symbol: "btc",
					Name:   "Bitcoin",
					Platforms: map[string]string{
						"solana":              "solbtc",
						"binance-smart-chain": "0xbtc-bsc",
					},
				},
			},
			supportedPlatforms: []string{"ethereum", "polygon-pos"},
			want:               []Token{},
		},
		{
			name: "some tokens without any platforms",
			tokens: []Token{
				{
					ID:        "bitcoin",
					Symbol:    "btc",
					Name:      "Bitcoin",
					Platforms: map[string]string{},
				},
				{
					ID:     "ethereum",
					Symbol: "eth",
					Name:   "Ethereum",
					Platforms: map[string]string{
						"ethereum": "0xeth",
					},
				},
			},
			supportedPlatforms: []string{"ethereum"},
			want: []Token{
				{
					ID:     "ethereum",
					Symbol: "eth",
					Name:   "Ethereum",
					Platforms: map[string]string{
						"ethereum": "0xeth",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterTokensByPlatform(tt.tokens, tt.supportedPlatforms)

			if len(got) != len(tt.want) {
				t.Errorf("FilterTokensByPlatform() returned %d tokens, want %d tokens", len(got), len(tt.want))
				return
			}

			// Compare each token
			for i, wantToken := range tt.want {
				gotToken := got[i]

				// Check token properties
				if gotToken.ID != wantToken.ID ||
					gotToken.Symbol != wantToken.Symbol ||
					gotToken.Name != wantToken.Name {
					t.Errorf("Token %d properties don't match: got %+v, want %+v", i, gotToken, wantToken)
				}

				// Check platforms
				if !reflect.DeepEqual(gotToken.Platforms, wantToken.Platforms) {
					t.Errorf("Token %d platforms don't match: got %v, want %v", i, gotToken.Platforms, wantToken.Platforms)
				}
			}
		})
	}
}
