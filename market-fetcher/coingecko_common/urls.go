package coingecko_common

import (
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/proxy-common/apikeys"
)

// GetApiBaseUrl returns the appropriate API URL based on the key type and config
func GetApiBaseUrl(cfg *config.Config, keyType apikeys.KeyType) string {
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
