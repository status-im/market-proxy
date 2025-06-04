package config

import "time"

type CoingeckoCoinslistFetcher struct {
	UpdateInterval     time.Duration `yaml:"update_interval"`
	SupportedPlatforms []string      `yaml:"supported_platforms"`
}
