package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	CoinGeckoFetcher struct {
		UpdateIntervalMs int    `yaml:"update_interval_ms"`
		TokensFile       string `yaml:"tokens_file"`
		Limit            int    `yaml:"limit"`
		RequestDelayMs   int    `yaml:"request_delay_ms"` // Delay between requests in milliseconds
	} `yaml:"coingecko_fetcher"`
}

type APITokens struct {
	Tokens     []string `json:"api_tokens"`
	DemoTokens []string `json:"demo_api_tokens,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	return &config, err
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
