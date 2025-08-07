package config

import "time"

type CoinslistFetcherConfig struct {
	UpdateInterval     time.Duration `yaml:"update_interval"`
	SupportedPlatforms []string      `yaml:"supported_platforms"`
}
