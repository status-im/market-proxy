package coingecko_common

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
)

// KeyType defines the API key type
type KeyType int

const (
	// NoKey means no API key is available
	NoKey KeyType = iota
	// ProKey means using a Pro API key
	ProKey
	// DemoKey means using a demo API key
	DemoKey
)

// APIKey represents an API key with its type
type APIKey struct {
	Key  string
	Type KeyType
}

// IAPIKeyManager defines the interface for API key management
type IAPIKeyManager interface {
	// GetAvailableKeys returns a list of available API keys, including:
	// - All Pro keys that are not in backoff
	// - If there's only one Pro key, it's included even if in backoff
	// - All Demo keys that are not in backoff
	// - A "no key" (empty key) entry at the end of the list
	GetAvailableKeys() []APIKey

	// MarkKeyAsFailed marks a key as failed, which will put it in backoff
	MarkKeyAsFailed(key string)
}

// APIKeyManager implements IAPIKeyManager for CoinGecko
type APIKeyManager struct {
	apiTokens   *config.APITokens
	rand        *rand.Rand
	lastFailed  map[string]time.Time // Stores the time of the last failure for each key
	backoffTime time.Duration        // Backoff duration before retrying a failed key
	mu          sync.RWMutex
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(apiTokens *config.APITokens) *APIKeyManager {
	return &APIKeyManager{
		apiTokens:   apiTokens,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		lastFailed:  make(map[string]time.Time),
		backoffTime: 5 * time.Minute,
	}
}

// isKeyInBackoff checks if a key is currently in backoff period (private implementation)
func (m *APIKeyManager) isKeyInBackoff(key string) bool {
	if key == "" {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if lastFailTime, exists := m.lastFailed[key]; exists {
		return time.Since(lastFailTime) < m.backoffTime
	}

	return false
}

// getKeysOfType returns all keys of a specific type (private implementation)
func (m *APIKeyManager) getKeysOfType(keyType KeyType) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.apiTokens == nil {
		return []string{}
	}

	switch keyType {
	case ProKey:
		return append([]string{}, m.apiTokens.Tokens...)
	case DemoKey:
		return append([]string{}, m.apiTokens.DemoTokens...)
	}

	return []string{}
}

// GetAvailableKeys returns a list of available API keys based on the specified logic
func (m *APIKeyManager) GetAvailableKeys() []APIKey {
	availableKeys := []APIKey{}

	proKeys := m.getKeysOfType(ProKey)

	// If there's exactly one Pro key, include it even if it's in backoff
	if len(proKeys) == 1 {
		availableKeys = append(availableKeys, APIKey{Key: proKeys[0], Type: ProKey})
	} else if len(proKeys) > 1 {
		// For multiple Pro keys, include only those not in backoff
		for _, key := range proKeys {
			if !m.isKeyInBackoff(key) {
				availableKeys = append(availableKeys, APIKey{Key: key, Type: ProKey})
			}
		}
	}

	// Add available Demo keys (not in backoff)
	demoKeys := m.getKeysOfType(DemoKey)
	for _, key := range demoKeys {
		if !m.isKeyInBackoff(key) {
			availableKeys = append(availableKeys, APIKey{Key: key, Type: DemoKey})
		}
	}

	// Always add the "no key" option at the end of the list
	availableKeys = append(availableKeys, APIKey{Key: "", Type: NoKey})

	return availableKeys
}

// MarkKeyAsFailed marks a key as non-working for some time
func (m *APIKeyManager) MarkKeyAsFailed(key string) {
	if key == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastFailed[key] = time.Now()
	log.Printf("APIKeyManager: Marked key as failed for %v", m.backoffTime)
}
