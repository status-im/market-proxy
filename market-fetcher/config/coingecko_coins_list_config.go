package config

type CoingeckoCoinslistFetcher struct {
	UpdateIntervalMs   int      `yaml:"update_interval_ms"`
	SupportedPlatforms []string `yaml:"supported_platforms"`
}
