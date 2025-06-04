package config

import (
	"io/ioutil"
	"log"

	"github.com/status-im/market-proxy/cache"
	"gopkg.in/yaml.v2"
)

type Config struct {
	CoingeckoLeaderboard CoingeckoLeaderboardFetcher `yaml:"coingecko_leaderboard"`
	TokensFetcher        CoingeckoCoinslistFetcher   `yaml:"coingecko_coinslist"`
	TokensFile           string                      `yaml:"tokens_file"`
	APITokens            *APITokens
	Cache                cache.Config `yaml:"cache"`

	OverrideCoingeckoPublicURL string `yaml:"override_coingecko_public_url"`
	OverrideCoingeckoProURL    string `yaml:"override_coingecko_pro_url"`
	OverrideBinanceWSURL       string `yaml:"override_binance_wsurl"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// Set default cache config if not provided
	if config.Cache.GoCache.DefaultExpiration == 0 && config.Cache.GoCache.CleanupInterval == 0 {
		config.Cache = cache.DefaultCacheConfig()
	}

	apiTokens, err := LoadAPITokens(config.TokensFile)
	if err != nil {
		log.Printf("Warning: Error loading API tokens from %s: %v. Using public API without authentication.",
			config.TokensFile, err)
		config.APITokens = &APITokens{Tokens: []string{}}
	} else {
		config.APITokens = apiTokens
	}

	return &config, nil
}
