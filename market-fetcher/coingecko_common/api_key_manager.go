package coingecko_common

import (
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

// NewAPIKeyManager creates a CoinGecko API key manager with Pro → Demo → NoKey priority.
func NewAPIKeyManager(t *config.APITokens) *apikeys.APIKeyManager {
	return apikeys.NewAPIKeyManager(
		tokensProvider{t: t},
		[]apikeys.KeyType{ProKey, DemoKey, NoKey},
		5*time.Minute,
	)
}
