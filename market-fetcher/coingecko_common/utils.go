package coingecko_common

import (
	"log"

	"github.com/status-im/market-proxy/config"
)

// GetApiBaseUrl возвращает нужный API URL в зависимости от типа ключа и конфига
func GetApiBaseUrl(cfg *config.Config, keyType KeyType) string {
	if keyType == ProKey {
		log.Printf("CoinGecko: Using Pro API URL based on key type")
		if cfg.OverrideCoingeckoProURL != "" {
			log.Printf("CoinGecko: Using overridden Pro API URL: %s", cfg.OverrideCoingeckoProURL)
			return cfg.OverrideCoingeckoProURL
		}
		return COINGECKO_PRO_URL
	}
	log.Printf("CoinGecko: Using Public API URL based on key type")
	if cfg.OverrideCoingeckoPublicURL != "" {
		log.Printf("CoinGecko: Using overridden public API URL: %s", cfg.OverrideCoingeckoPublicURL)
		return cfg.OverrideCoingeckoPublicURL
	}
	return COINGECKO_PUBLIC_URL
}
