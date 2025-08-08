package coingecko_common

import (
	"math"
	"net/url"
	"strings"
	"sync"

	"github.com/status-im/market-proxy/config"
	"golang.org/x/time/rate"
)

// IRateLimiterManager provides a way to get a rate limiter for a request URL
//
//go:generate mockgen -destination=mocks/rate_limiter_manager.go . IRateLimiterManager
type IRateLimiterManager interface {
	GetLimiterForURL(u *url.URL) *rate.Limiter
	SetConfig(cfg config.APIKeyConfig)
}

// RateLimiterManager manages per-key rate limiters using APIKeyConfig
type RateLimiterManager struct {
	mu           sync.RWMutex
	keyToLimiter map[string]*rate.Limiter
	config       config.APIKeyConfig
}

var (
	managerOnce   sync.Once
	globalManager *RateLimiterManager
)

// Defaults in requests per minute, used when config is not provided
const (
	defaultProRPM   = 500
	defaultDemoRPM  = 30
	defaultNoKeyRPM = 30
)

// GetRateLimiterManagerInstance returns the global singleton RateLimiterManager instance
func GetRateLimiterManagerInstance() *RateLimiterManager {
	managerOnce.Do(func() {
		globalManager = &RateLimiterManager{
			keyToLimiter: make(map[string]*rate.Limiter),
			config:       config.APIKeyConfig{},
		}
	})
	return globalManager
}

// SetConfig applies a new APIKeyConfig and rebuilds limiters for types with changed settings.
// If Burst is not provided (>0) in the new config for a type, the existing burst is not reused.
func (m *RateLimiterManager) SetConfig(newCfg config.APIKeyConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldCfg := m.config
	m.config = newCfg

	// Detect changes per key type
	type change struct{ rpm, burst bool }
	changed := map[KeyType]change{
		ProKey:  {rpm: oldCfg.Pro.RateLimitPerMinute != newCfg.Pro.RateLimitPerMinute, burst: oldCfg.Pro.Burst != newCfg.Pro.Burst},
		DemoKey: {rpm: oldCfg.Demo.RateLimitPerMinute != newCfg.Demo.RateLimitPerMinute, burst: oldCfg.Demo.Burst != newCfg.Demo.Burst},
		NoKey:   {rpm: oldCfg.NoKey.RateLimitPerMinute != newCfg.NoKey.RateLimitPerMinute, burst: oldCfg.NoKey.Burst != newCfg.NoKey.Burst},
	}

	for mapKey := range m.keyToLimiter {
		kType := m.parseKeyType(mapKey)
		if ch, ok := changed[kType]; ok && (ch.rpm || ch.burst) {
			limit := m.limitForTypeLocked(kType)
			burst := m.burstForTypeWithDefaultFromLimit(kType, limit)
			m.keyToLimiter[mapKey] = rate.NewLimiter(limit, burst)
		}
	}
}

// GetLimiterForURL inspects the URL to determine key and type and returns appropriate limiter
func (m *RateLimiterManager) GetLimiterForURL(u *url.URL) *rate.Limiter {
	if m == nil || u == nil {
		return nil
	}

	query := u.Query()

	// Prefer explicit key params
	if v := query.Get("x_cg_pro_api_key"); v != "" {
		return m.getLimiterForKey(v, ProKey)
	}
	if v := query.Get("x_cg_demo_api_key"); v != "" {
		return m.getLimiterForKey(v, DemoKey)
	}

	// Apply public limiter only for known CoinGecko hosts
	host := u.Hostname()
	if host == "api.coingecko.com" || host == "pro-api.coingecko.com" {
		return m.getLimiterForKey("", NoKey)
	}

	// No limiter for unrelated hosts
	return nil
}

// getLimiterForKey returns a limiter for a given api key and type, creating it if missing
func (m *RateLimiterManager) getLimiterForKey(key string, keyType KeyType) *rate.Limiter {
	mapKey := m.limiterMapKey(key, keyType)

	m.mu.RLock()
	if lim, ok := m.keyToLimiter[mapKey]; ok {
		m.mu.RUnlock()
		return lim
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if lim, ok := m.keyToLimiter[mapKey]; ok {
		return lim
	}

	limit := m.limitForTypeLocked(keyType)
	burst := m.burstForTypeWithDefaultFromLimit(keyType, limit)
	limiter := rate.NewLimiter(limit, burst)
	m.keyToLimiter[mapKey] = limiter
	return limiter
}

func (m *RateLimiterManager) limiterMapKey(key string, keyType KeyType) string {
	return "type:" + m.keyTypeString(keyType) + "|key:" + key
}

func (m *RateLimiterManager) keyTypeString(keyType KeyType) string {
	switch keyType {
	case ProKey:
		return "pro"
	case DemoKey:
		return "demo"
	case NoKey:
		return "none"
	default:
		return "unknown"
	}
}

func (m *RateLimiterManager) parseKeyType(mapKey string) KeyType {
	if strings.HasPrefix(mapKey, "type:") {
		rest := strings.TrimPrefix(mapKey, "type:")
		idx := strings.Index(rest, "|")
		if idx > 0 {
			t := rest[:idx]
			switch t {
			case "pro":
				return ProKey
			case "demo":
				return DemoKey
			case "none":
				return NoKey
			}
		}
	}
	return NoKey
}

func (m *RateLimiterManager) limitForTypeLocked(keyType KeyType) rate.Limit {
	rpm := 0
	switch keyType {
	case ProKey:
		rpm = m.config.Pro.RateLimitPerMinute
		if rpm <= 0 {
			rpm = defaultProRPM
		}
	case DemoKey:
		rpm = m.config.Demo.RateLimitPerMinute
		if rpm <= 0 {
			rpm = defaultDemoRPM
		}
	case NoKey:
		rpm = m.config.NoKey.RateLimitPerMinute
		if rpm <= 0 {
			rpm = defaultNoKeyRPM
		}
	default:
		rpm = defaultNoKeyRPM
	}
	return rate.Limit(float64(rpm) / 60.0)
}

func (m *RateLimiterManager) burstForTypeWithDefaultFromLimit(keyType KeyType, limit rate.Limit) int {
	switch keyType {
	case ProKey:
		if m.config.Pro.Burst > 0 {
			return m.config.Pro.Burst
		}
	case DemoKey:
		if m.config.Demo.Burst > 0 {
			return m.config.Demo.Burst
		}
	case NoKey:
		if m.config.NoKey.Burst > 0 {
			return m.config.NoKey.Burst
		}
	}
	return defaultBurstForLimit(limit)
}

func defaultBurstForLimit(limit rate.Limit) int {
	if limit <= 1.0 {
		return 1
	}
	return int(math.Ceil(float64(limit)))
}
