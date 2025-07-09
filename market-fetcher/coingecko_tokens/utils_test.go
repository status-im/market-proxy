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
			name: "native tokens are included even without platforms",
			tokens: []Token{
				{
					ID:        "ethereum",
					Symbol:    "eth",
					Name:      "Ethereum",
					Platforms: map[string]string{},
				},
				{
					ID:        "polygon-pos",
					Symbol:    "matic",
					Name:      "Polygon",
					Platforms: map[string]string{},
				},
				{
					ID:     "bitcoin",
					Symbol: "btc",
					Name:   "Bitcoin",
					Platforms: map[string]string{
						"solana": "solbtc",
					},
				},
			},
			supportedPlatforms: []string{"ethereum", "polygon-pos"},
			want: []Token{
				{
					ID:        "ethereum",
					Symbol:    "eth",
					Name:      "Ethereum",
					Platforms: map[string]string{},
				},
				{
					ID:        "polygon-pos",
					Symbol:    "matic",
					Name:      "Polygon",
					Platforms: map[string]string{},
				},
			},
		},
		{
			name: "native token with supported platforms",
			tokens: []Token{
				{
					ID:     "ethereum",
					Symbol: "eth",
					Name:   "Ethereum",
					Platforms: map[string]string{
						"ethereum":    "native",
						"polygon-pos": "0xeth-poly",
						"solana":      "eth-sol",
					},
				},
			},
			supportedPlatforms: []string{"ethereum", "polygon-pos"},
			want: []Token{
				{
					ID:     "ethereum",
					Symbol: "eth",
					Name:   "Ethereum",
					Platforms: map[string]string{
						"ethereum":    "native",
						"polygon-pos": "0xeth-poly",
					},
				},
			},
		},
		{
			name: "mix of native tokens and platform tokens",
			tokens: []Token{
				{
					ID:        "ethereum",
					Symbol:    "eth",
					Name:      "Ethereum",
					Platforms: map[string]string{},
				},
				{
					ID:     "usdc",
					Symbol: "usdc",
					Name:   "USD Coin",
					Platforms: map[string]string{
						"ethereum":    "0xa0b86a33e6776b1e0e729c3b87c3c8c3",
						"polygon-pos": "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
					},
				},
				{
					ID:     "bitcoin",
					Symbol: "btc",
					Name:   "Bitcoin",
					Platforms: map[string]string{
						"solana": "solbtc",
					},
				},
			},
			supportedPlatforms: []string{"ethereum"},
			want: []Token{
				{
					ID:        "ethereum",
					Symbol:    "eth",
					Name:      "Ethereum",
					Platforms: map[string]string{},
				},
				{
					ID:     "usdc",
					Symbol: "usdc",
					Name:   "USD Coin",
					Platforms: map[string]string{
						"ethereum": "0xa0b86a33e6776b1e0e729c3b87c3c8c3",
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
			name: "no matching platforms or native tokens",
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

func TestCountTokensByPlatform(t *testing.T) {
	tests := []struct {
		name   string
		tokens []Token
		want   map[string]int
	}{
		{
			name: "count tokens across multiple platforms",
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
				{
					ID:     "ethereum",
					Symbol: "eth",
					Name:   "Ethereum",
					Platforms: map[string]string{
						"polygon-pos": "0xeth-poly",
						"solana":      "eth-sol",
					},
				},
				{
					ID:     "usdc",
					Symbol: "usdc",
					Name:   "USD Coin",
					Platforms: map[string]string{
						"ethereum": "0xa0b86a33e6776b1e0e729c3b87c3c8c3",
					},
				},
			},
			want: map[string]int{
				"ethereum":    2,
				"polygon-pos": 2,
				"solana":      1,
			},
		},
		{
			name:   "empty tokens list",
			tokens: []Token{},
			want:   map[string]int{},
		},
		{
			name: "tokens without platforms",
			tokens: []Token{
				{
					ID:        "bitcoin",
					Symbol:    "btc",
					Name:      "Bitcoin",
					Platforms: map[string]string{},
				},
			},
			want: map[string]int{},
		},
		{
			name: "single platform with multiple tokens",
			tokens: []Token{
				{
					ID:     "usdc",
					Symbol: "usdc",
					Name:   "USD Coin",
					Platforms: map[string]string{
						"ethereum": "0xa0b86a33e6776b1e0e729c3b87c3c8c3",
					},
				},
				{
					ID:     "usdt",
					Symbol: "usdt",
					Name:   "Tether",
					Platforms: map[string]string{
						"ethereum": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					},
				},
			},
			want: map[string]int{
				"ethereum": 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountTokensByPlatform(tt.tokens)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CountTokensByPlatform() = %v, want %v", got, tt.want)
			}
		})
	}
}
