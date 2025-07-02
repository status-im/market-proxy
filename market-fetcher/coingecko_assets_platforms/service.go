package coingecko_assets_platforms

import (
	"context"
	"fmt"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
)

type Service struct {
	config        *config.Config
	client        APIClient
	metricsWriter *metrics.MetricsWriter
}

type APIClient interface {
	FetchAssetsPlatforms(params AssetsPlatformsParams) ([]byte, error)
	Healthy() bool
}

func NewService(config *config.Config) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServicePlatforms)
	client := NewCoinGeckoClient(config)

	return &Service{
		config:        config,
		client:        client,
		metricsWriter: metricsWriter,
	}
}

func (s *Service) Start(ctx context.Context) error {
	return nil
}

func (s *Service) Stop() {
}

func (s *Service) AssetsPlatforms(params AssetsPlatformsParams) ([]byte, error) {
	data, err := s.client.FetchAssetsPlatforms(params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assets platforms: %w", err)
	}

	return data, nil
}

func (s *Service) Healthy() bool {
	if s.client != nil {
		return s.client.Healthy()
	}
	return false
}
