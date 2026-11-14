package coingecko_common

import (
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/proxy-common/apikeys"
)

// Key type constants (values must match rate_limiter_manager / proxy-common conventions).
const (
	NoKey   apikeys.KeyType = 0
	ProKey  apikeys.KeyType = 1
	DemoKey apikeys.KeyType = 2
)

// keyBackoff is how long a failed key is left alone.
const keyBackoff = 5 * time.Minute

// APIKeyManager is the CoinGecko API key manager: proxy-common's manager with
// Pro → Demo → NoKey priority, plus one rule of its own for demo keys.
//
// proxy-common keeps a key available while it is in backoff if it is the only
// one of its type. That suits a Pro key, which there is nothing better to
// replace with, but not a demo key: a request would spend its retries on a
// rate-limited demo key before falling back to no key, on every request. So a
// demo key in backoff is skipped even when it is the only one.
type APIKeyManager struct {
	keys    *apikeys.APIKeyManager
	backoff time.Duration

	mu         sync.RWMutex
	lastFailed map[string]time.Time
}

// NewAPIKeyManager creates a CoinGecko API key manager.
func NewAPIKeyManager(t *config.APITokens) *APIKeyManager {
	return newAPIKeyManager(t, keyBackoff)
}

func newAPIKeyManager(t *config.APITokens, backoff time.Duration) *APIKeyManager {
	return &APIKeyManager{
		keys: apikeys.NewAPIKeyManager(
			tokensProvider{t: t},
			[]apikeys.KeyType{ProKey, DemoKey, NoKey},
			backoff,
		),
		backoff:    backoff,
		lastFailed: make(map[string]time.Time),
	}
}

// GetAvailableKeys returns the keys to try, in order:
//   - Pro keys that are not in backoff; a single Pro key even if it is
//   - Demo keys that are not in backoff
//   - the "no key" entry
func (m *APIKeyManager) GetAvailableKeys() []apikeys.APIKey {
	keys := m.keys.GetAvailableKeys()

	available := keys[:0]
	for _, key := range keys {
		if key.Type == DemoKey && m.isInBackoff(key.Key) {
			continue
		}
		available = append(available, key)
	}
	return available
}

// MarkKeyAsFailed puts a key in backoff.
func (m *APIKeyManager) MarkKeyAsFailed(key string) {
	if key != "" {
		m.mu.Lock()
		m.lastFailed[key] = time.Now()
		m.mu.Unlock()
	}
	m.keys.MarkKeyAsFailed(key)
}

func (m *APIKeyManager) isInBackoff(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	failedAt, failed := m.lastFailed[key]
	return failed && time.Since(failedAt) < m.backoff
}
