package config

import (
	"encoding/json"
	"os"
	"time"
)

type LeaderboardFetcherConfig struct {
	TopMarketsUpdateInterval time.Duration `yaml:"top_markets_update_interval"`
	TopMarketsLimit          int           `yaml:"top_markets_limit"`
	Currency                 string        `yaml:"currency"`                   // Currency for market data
	TopPricesUpdateInterval  time.Duration `yaml:"top_prices_update_interval"` // Interval for price updates
	TopPricesLimit           int           `yaml:"top_prices_limit"`           // Limit for top tokens prices
}

type APITokens struct {
	Tokens     []string `json:"api_tokens"`
	DemoTokens []string `json:"demo_api_tokens,omitempty"`
}

func LoadAPITokens(filename string) (*APITokens, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// File doesn't exist, return empty tokens
		return &APITokens{Tokens: []string{}}, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var tokens APITokens
	err = json.Unmarshal(data, &tokens)
	return &tokens, err
}
