package coingecko_common

import (
	"errors"
	"testing"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/proxy-common/apikeys"
	"github.com/stretchr/testify/assert"
)

func TestGetApiBaseUrl(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		keyType     apikeys.KeyType
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

// Mock APIKeyManager for testing
type mockAPIKeyManager struct {
	keys       []apikeys.APIKey
	failedKeys []string
}

func (m *mockAPIKeyManager) GetAvailableKeys() []apikeys.APIKey {
	return m.keys
}

func (m *mockAPIKeyManager) MarkKeyAsFailed(key string) {
	m.failedKeys = append(m.failedKeys, key)
}

func TestTryWithKeys(t *testing.T) {
	tests := []struct {
		name             string
		keys             []apikeys.APIKey
		executorBehavior func(apiKey apikeys.APIKey) (interface{}, bool, error)
		expectedResult   interface{}
		expectedError    string
		expectedFailed   []string
	}{
		{
			name: "Success on first key",
			keys: []apikeys.APIKey{
				{Key: "key1", Type: ProKey},
				{Key: "key2", Type: DemoKey},
			},
			executorBehavior: func(apiKey apikeys.APIKey) (interface{}, bool, error) {
				return "success", true, nil
			},
			expectedResult: "success",
			expectedError:  "",
			expectedFailed: []string{},
		},
		{
			name: "Success on second key",
			keys: []apikeys.APIKey{
				{Key: "key1", Type: ProKey},
				{Key: "key2", Type: DemoKey},
			},
			executorBehavior: func(apiKey apikeys.APIKey) (interface{}, bool, error) {
				if apiKey.Key == "key1" {
					return nil, false, errors.New("first key failed")
				}
				return "success", true, nil
			},
			expectedResult: "success",
			expectedError:  "",
			expectedFailed: []string{"key1"},
		},
		{
			name: "All keys fail",
			keys: []apikeys.APIKey{
				{Key: "key1", Type: ProKey},
				{Key: "", Type: NoKey},
			},
			executorBehavior: func(apiKey apikeys.APIKey) (interface{}, bool, error) {
				return nil, false, errors.New("failed")
			},
			expectedResult: nil,
			expectedError:  "all API keys failed, last error: failed",
			expectedFailed: []string{"key1"}, // Empty keys should not be marked as failed
		},
		{
			name: "No keys available",
			keys: []apikeys.APIKey{},
			executorBehavior: func(apiKey apikeys.APIKey) (interface{}, bool, error) {
				return "success", true, nil
			},
			expectedResult: nil,
			expectedError:  "all API keys failed, last error: <nil>",
			expectedFailed: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKeyManager := &mockAPIKeyManager{
				keys:       tt.keys,
				failedKeys: []string{},
			}

			onFailed := apikeys.CreateFailCallback(mockKeyManager)
			result, err := apikeys.TryWithKeys(mockKeyManager.keys, "TEST", tt.executorBehavior, onFailed)

			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedFailed, mockKeyManager.failedKeys)
		})
	}
}
