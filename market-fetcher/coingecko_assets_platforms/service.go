package coingecko_assets_platforms

import (
	"context"

	"github.com/status-im/market-proxy/config"
)

type IAPIClient interface {
	FetchAssetsPlatforms(params AssetsPlatformsParams) (AssetsPlatformsResponse, error)
	Healthy() bool
}

type Service struct {
	config *config.Config
	client IAPIClient
}

func NewService(config *config.Config) *Service {
	client := NewCoinGeckoClient(config)
	return &Service{
		config: config,
		client: client,
	}
}

func (s *Service) Start(ctx context.Context) error {
	return nil
}

func (s *Service) Stop() {
}

func (s *Service) AssetsPlatforms(params AssetsPlatformsParams) (AssetsPlatformsResponse, error) {
	return s.client.FetchAssetsPlatforms(params)
}

func (s *Service) Healthy() bool {
	if s.client != nil {
		return s.client.Healthy()
	}
	return false
}
