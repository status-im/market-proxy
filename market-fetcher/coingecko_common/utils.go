package coingecko_common

import (
	"fmt"
	"log"

	"github.com/status-im/market-proxy/config"
)

// RequestExecutor represents a function that attempts to execute a request with a given API key
// It returns the response data, success flag, and any error
type RequestExecutor func(apiKey APIKey) (interface{}, bool, error)

// OnFailedCallback represents a function that is called when an API key fails
type OnFailedCallback func(apiKey APIKey)

// CreateFailCallback creates a callback that marks keys as failed
func CreateFailCallback(keyManager IAPIKeyManager) OnFailedCallback {
	return func(apiKey APIKey) {
		if apiKey.Key != "" {
			log.Printf("Marking key as failed and adding to backoff")
			keyManager.MarkKeyAsFailed(apiKey.Key)
		}
	}
}

// TryWithKeys attempts to execute a request function with each available API key until one succeeds
func TryWithKeys(availableKeys []APIKey, logPrefix string, executor RequestExecutor, onFailed OnFailedCallback) (interface{}, error) {
	var lastError error

	for _, apiKey := range availableKeys {
		result, success, err := executor(apiKey)

		if success {
			return result, nil
		}

		if err != nil {
			log.Printf("%s: Request failed with key type %v: %v", logPrefix, apiKey.Type, err)

			// Call the onFailed callback
			if onFailed != nil {
				onFailed(apiKey)
			}

			lastError = err
		}
	}

	return nil, fmt.Errorf("all API keys failed, last error: %v", lastError)
}

// GetApiBaseUrl returns the appropriate API URL based on the key type and config
func GetApiBaseUrl(cfg *config.Config, keyType KeyType) string {
	if keyType == ProKey {
		if cfg.OverrideCoingeckoProURL != "" {
			return cfg.OverrideCoingeckoProURL
		}
		return COINGECKO_PRO_URL
	}
	if cfg.OverrideCoingeckoPublicURL != "" {
		return cfg.OverrideCoingeckoPublicURL
	}
	return COINGECKO_PUBLIC_URL
}
