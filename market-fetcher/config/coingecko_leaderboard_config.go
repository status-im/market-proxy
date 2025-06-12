package config

import (
	"encoding/json"
	"os"
	"time"
)

type CoingeckoLeaderboardFetcher struct {
	UpdateInterval       time.Duration `yaml:"update_interval"`
	Limit                int           `yaml:"limit"`
	RequestDelay         time.Duration `yaml:"request_delay"`          // Delay between requests
	PricesUpdateInterval time.Duration `yaml:"prices_update_interval"` // Interval for price updates
	TopTokensLimit       int           `yaml:"top_tokens_limit"`       // Limit for top tokens prices
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
