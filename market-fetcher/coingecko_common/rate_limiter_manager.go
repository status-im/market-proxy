package coingecko_common

import (
	"net/url"
	"sync"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/proxy-common/apikeys"
	prl "github.com/status-im/proxy-common/ratelimit"
	"golang.org/x/time/rate"
)

// IRateLimiterManager provides a way to get a rate limiter for a request URL
//
//go:generate mockgen -destination=mocks/rate_limiter_manager.go . IRateLimiterManager
type IRateLimiterManager interface {
	GetLimiterForURL(u *url.URL) *rate.Limiter
	SetConfig(cfg config.APIKeyConfig)
}

// RateLimiterManager manages per-key rate limiters using APIKeyConfig.
// It delegates per-key *rate.Limiter storage to proxy-common/ratelimit while
// keeping CoinGecko URL parsing and type-specific default RPMs here.
type RateLimiterManager struct {
	mu     sync.Mutex
	inner  *prl.RateLimiterManager
	config config.APIKeyConfig
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

// newRateLimiterManager is used by tests; production uses the singleton.
func newRateLimiterManager(cfg config.APIKeyConfig) *RateLimiterManager {
	return &RateLimiterManager{
		inner:  prl.NewRateLimiterManager(buildLibCfg(cfg)),
		config: cfg,
	}
}

// buildLibCfg maps our YAML config to proxy-common's map with CoinGecko defaults applied.
// Burst 0 in cfg means the library will use defaultBurstForLimit from the effective rate.
func buildLibCfg(cfg config.APIKeyConfig) map[apikeys.KeyType]prl.RateLimit {
	rpmPro := cfg.Pro.RateLimitPerMinute
	if rpmPro <= 0 {
		rpmPro = defaultProRPM
	}
	rpmDemo := cfg.Demo.RateLimitPerMinute
	if rpmDemo <= 0 {
		rpmDemo = defaultDemoRPM
	}
	rpmNoKey := cfg.NoKey.RateLimitPerMinute
	if rpmNoKey <= 0 {
		rpmNoKey = defaultNoKeyRPM
	}
	return map[apikeys.KeyType]prl.RateLimit{
		ProKey:  {RateLimitPerMinute: rpmPro, Burst: cfg.Pro.Burst},
		DemoKey: {RateLimitPerMinute: rpmDemo, Burst: cfg.Demo.Burst},
		NoKey:   {RateLimitPerMinute: rpmNoKey, Burst: cfg.NoKey.Burst},
	}
}

// GetRateLimiterManagerInstance returns the global singleton RateLimiterManager instance
func GetRateLimiterManagerInstance() *RateLimiterManager {
	managerOnce.Do(func() {
		empty := config.APIKeyConfig{}
		globalManager = &RateLimiterManager{
			inner:  prl.NewRateLimiterManager(buildLibCfg(empty)),
			config: empty,
		}
	})
	return globalManager
}

// SetConfig applies a new APIKeyConfig and propagates it to the underlying rate limit manager.
// All in-memory limiters are reset (per proxy-common/ratelimit.SetConfig).
func (m *RateLimiterManager) SetConfig(newCfg config.APIKeyConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = newCfg
	if m.inner != nil {
		m.inner.SetConfig(buildLibCfg(newCfg))
	}
}

// GetLimiterForURL inspects the URL to determine key and type and returns appropriate limiter
func (m *RateLimiterManager) GetLimiterForURL(u *url.URL) *rate.Limiter {
	if m == nil || u == nil {
		return nil
	}
	if m.inner == nil {
		return nil
	}

	query := u.Query()

	// Prefer explicit key params
	if v := query.Get("x_cg_pro_api_key"); v != "" {
		return m.inner.GetLimiter(v, ProKey)
	}
	if v := query.Get("x_cg_demo_api_key"); v != "" {
		return m.inner.GetLimiter(v, DemoKey)
	}

	// Apply public limiter only for known CoinGecko hosts
	host := u.Hostname()
	if host == "api.coingecko.com" || host == "pro-api.coingecko.com" {
		return m.inner.GetLimiter("", NoKey)
	}

	// No limiter for unrelated hosts
	return nil
}
