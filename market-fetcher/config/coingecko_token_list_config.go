package config

import "time"

type TokenListFetcherConfig struct {
	UpdateInterval     time.Duration `yaml:"update_interval"`
	SupportedPlatforms []string      `yaml:"supported_platforms"`
}
