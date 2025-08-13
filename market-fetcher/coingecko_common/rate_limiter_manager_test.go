package coingecko_common

import (
	"net/url"
	"testing"

	"github.com/status-im/market-proxy/config"
	"golang.org/x/time/rate"
)

func TestRateLimiterManager_GetLimiterForURL(t *testing.T) {
	// Create manager with test config
	manager := &RateLimiterManager{
		keyToLimiter: make(map[string]*rate.Limiter),
		config: config.APIKeyConfig{
			Pro: config.RateLimit{
				RateLimitPerMinute: 300,
				Burst:              10,
			},
			Demo: config.RateLimit{
				RateLimitPerMinute: 60,
				Burst:              2,
			},
			NoKey: config.RateLimit{
				RateLimitPerMinute: 30,
				Burst:              1,
			},
		},
	}

	tests := []struct {
		name            string
		url             string
		expectedLimiter bool
		description     string
	}{
		{
			name:            "ProAPIKey",
			url:             "https://pro-api.coingecko.com/api/v3/coins/markets?x_cg_pro_api_key=test-pro-key",
			expectedLimiter: true,
			description:     "Should return limiter for pro API key",
		},
		{
			name:            "DemoAPIKey",
			url:             "https://api.coingecko.com/api/v3/coins/markets?x_cg_demo_api_key=test-demo-key",
			expectedLimiter: true,
			description:     "Should return limiter for demo API key",
		},
		{
			name:            "NoKeyPublicAPI",
			url:             "https://api.coingecko.com/api/v3/coins/markets",
			expectedLimiter: true,
			description:     "Should return NoKey limiter for public CoinGecko API",
		},
		{
			name:            "NoKeyProHost",
			url:             "https://pro-api.coingecko.com/api/v3/coins/markets",
			expectedLimiter: true,
			description:     "Should return NoKey limiter for pro host without key",
		},
		{
			name:            "NonCoinGeckoHost",
			url:             "https://example.com/api/data",
			expectedLimiter: false,
			description:     "Should not return limiter for non-CoinGecko hosts",
		},
		{
			name:            "ProKeyPreferredOverDemo",
			url:             "https://api.coingecko.com/api/v3/coins/markets?x_cg_pro_api_key=pro-key&x_cg_demo_api_key=demo-key",
			expectedLimiter: true,
			description:     "Should prefer pro key when both are present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			limiter := manager.GetLimiterForURL(u)

			if tt.expectedLimiter && limiter == nil {
				t.Errorf("Expected limiter but got nil. %s", tt.description)
			}
			if !tt.expectedLimiter && limiter != nil {
				t.Errorf("Expected no limiter but got one. %s", tt.description)
			}
		})
	}
}

func TestRateLimiterManager_GetLimiterForURL_NilChecks(t *testing.T) {
	manager := &RateLimiterManager{
		keyToLimiter: make(map[string]*rate.Limiter),
		config:       config.APIKeyConfig{},
	}

	// Test nil URL
	limiter := manager.GetLimiterForURL(nil)
	if limiter != nil {
		t.Error("Expected nil limiter for nil URL")
	}

	// Test nil manager
	var nilManager *RateLimiterManager
	limiter = nilManager.GetLimiterForURL(&url.URL{})
	if limiter != nil {
		t.Error("Expected nil limiter for nil manager")
	}
}

func TestRateLimiterManager_SameLimiterForSameKey(t *testing.T) {
	manager := &RateLimiterManager{
		keyToLimiter: make(map[string]*rate.Limiter),
		config:       config.APIKeyConfig{},
	}

	// Create two URLs with the same pro key
	url1, _ := url.Parse("https://pro-api.coingecko.com/api/v3/coins/markets?x_cg_pro_api_key=same-key")
	url2, _ := url.Parse("https://pro-api.coingecko.com/api/v3/coins/list?x_cg_pro_api_key=same-key")

	limiter1 := manager.GetLimiterForURL(url1)
	limiter2 := manager.GetLimiterForURL(url2)

	if limiter1 == nil || limiter2 == nil {
		t.Fatal("Expected both limiters to be non-nil")
	}

	if limiter1 != limiter2 {
		t.Error("Expected same limiter instance for same API key")
	}
}

func TestRateLimiterManager_DifferentLimitersForDifferentKeys(t *testing.T) {
	manager := &RateLimiterManager{
		keyToLimiter: make(map[string]*rate.Limiter),
		config:       config.APIKeyConfig{},
	}

	// Create URLs with different keys
	urlPro, _ := url.Parse("https://pro-api.coingecko.com/api/v3/coins/markets?x_cg_pro_api_key=pro-key")
	urlDemo, _ := url.Parse("https://api.coingecko.com/api/v3/coins/markets?x_cg_demo_api_key=demo-key")
	urlNoKey, _ := url.Parse("https://api.coingecko.com/api/v3/coins/markets")

	limiterPro := manager.GetLimiterForURL(urlPro)
	limiterDemo := manager.GetLimiterForURL(urlDemo)
	limiterNoKey := manager.GetLimiterForURL(urlNoKey)

	if limiterPro == nil || limiterDemo == nil || limiterNoKey == nil {
		t.Fatal("Expected all limiters to be non-nil")
	}

	if limiterPro == limiterDemo || limiterPro == limiterNoKey || limiterDemo == limiterNoKey {
		t.Error("Expected different limiter instances for different key types")
	}
}

func TestRateLimiterManager_ConfiguredRates(t *testing.T) {
	// Test with specific configuration
	manager := &RateLimiterManager{
		keyToLimiter: make(map[string]*rate.Limiter),
		config: config.APIKeyConfig{
			Pro: config.RateLimit{
				RateLimitPerMinute: 600, // 10 rps
				Burst:              15,
			},
		},
	}

	urlPro, _ := url.Parse("https://pro-api.coingecko.com/api/v3/coins/markets?x_cg_pro_api_key=test-key")
	limiter := manager.GetLimiterForURL(urlPro)

	if limiter == nil {
		t.Fatal("Expected limiter to be non-nil")
	}

	// Check that burst is configured correctly
	if limiter.Burst() != 15 {
		t.Errorf("Expected burst of 15, got %d", limiter.Burst())
	}

	// Check that limit is approximately 10 per second (600/60)
	expectedLimit := float64(600) / 60.0
	actualLimit := float64(limiter.Limit())
	tolerance := 0.01
	if actualLimit < expectedLimit-tolerance || actualLimit > expectedLimit+tolerance {
		t.Errorf("Expected limit around %.2f, got %.2f", expectedLimit, actualLimit)
	}
}

func TestRateLimiterManager_DefaultRates(t *testing.T) {
	// Test with empty config to use defaults
	manager := &RateLimiterManager{
		keyToLimiter: make(map[string]*rate.Limiter),
		config:       config.APIKeyConfig{},
	}

	urlPro, _ := url.Parse("https://pro-api.coingecko.com/api/v3/coins/markets?x_cg_pro_api_key=test-key")
	urlDemo, _ := url.Parse("https://api.coingecko.com/api/v3/coins/markets?x_cg_demo_api_key=demo-key")
	urlNoKey, _ := url.Parse("https://api.coingecko.com/api/v3/coins/markets")

	limiterPro := manager.GetLimiterForURL(urlPro)
	limiterDemo := manager.GetLimiterForURL(urlDemo)
	limiterNoKey := manager.GetLimiterForURL(urlNoKey)

	// Check default rates (from constants in registry)
	expectedProRate := float64(defaultProRPM) / 60.0
	expectedDemoRate := float64(defaultDemoRPM) / 60.0
	expectedNoKeyRate := float64(defaultNoKeyRPM) / 60.0

	tolerance := 0.01

	if actual := float64(limiterPro.Limit()); actual < expectedProRate-tolerance || actual > expectedProRate+tolerance {
		t.Errorf("Expected pro rate around %.2f, got %.2f", expectedProRate, actual)
	}

	if actual := float64(limiterDemo.Limit()); actual < expectedDemoRate-tolerance || actual > expectedDemoRate+tolerance {
		t.Errorf("Expected demo rate around %.2f, got %.2f", expectedDemoRate, actual)
	}

	if actual := float64(limiterNoKey.Limit()); actual < expectedNoKeyRate-tolerance || actual > expectedNoKeyRate+tolerance {
		t.Errorf("Expected nokey rate around %.2f, got %.2f", expectedNoKeyRate, actual)
	}
}
