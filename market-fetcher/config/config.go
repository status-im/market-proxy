package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	CoingeckoLeaderboard CoingeckoLeaderboardFetcher `yaml:"coingecko_leaderboard"`
	TokensFetcher        CoingeckoCoinslistFetcher   `yaml:"coingecko_coinslist"`
	TokensFile           string                      `yaml:"tokens_file"`
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
