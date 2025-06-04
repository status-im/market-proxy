package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

type CoingeckoLeaderboardFetcher struct {
	UpdateInterval time.Duration `yaml:"update_interval"`
	Limit          int           `yaml:"limit"`
	RequestDelay   time.Duration `yaml:"request_delay"` // Delay between requests
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

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var tokens APITokens
	err = json.Unmarshal(data, &tokens)
	return &tokens, err
}
