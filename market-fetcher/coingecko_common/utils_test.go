package coingecko_common

import (
	"testing"

	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
)

func TestGetApiBaseUrl(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		keyType     KeyType
		expectedURL string
	}{
		{
			name:        "Pro key with default URL",
			cfg:         &config.Config{},
			keyType:     ProKey,
			expectedURL: COINGECKO_PRO_URL,
		},
		{
			name: "Pro key with overridden URL",
			cfg: &config.Config{
				OverrideCoingeckoProURL: "https://custom-pro.example.com",
			},
			keyType:     ProKey,
			expectedURL: "https://custom-pro.example.com",
		},
		{
			name:        "Public key with default URL",
			cfg:         &config.Config{},
			keyType:     NoKey,
			expectedURL: COINGECKO_PUBLIC_URL,
		},
		{
			name: "Public key with overridden URL",
			cfg: &config.Config{
				OverrideCoingeckoPublicURL: "https://custom-public.example.com",
			},
			keyType:     NoKey,
			expectedURL: "https://custom-public.example.com",
		},
		{
			name:        "Demo key with default URL",
			cfg:         &config.Config{},
			keyType:     DemoKey,
			expectedURL: COINGECKO_PUBLIC_URL,
		},
		{
			name: "Demo key with overridden public URL",
			cfg: &config.Config{
				OverrideCoingeckoPublicURL: "https://custom-public.example.com",
			},
			keyType:     DemoKey,
			expectedURL: "https://custom-public.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := GetApiBaseUrl(tt.cfg, tt.keyType)
			assert.Equal(t, tt.expectedURL, url)
		})
	}
}
