package coingecko_token_list

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/metrics"
)

type Service struct {
	config              *config.Config
	client              IClient
	metricsWriter       *metrics.MetricsWriter
	subscriptionManager *events.SubscriptionManager
	cache               sync.Map // map[string]*TokenListCache
	periodicUpdater     *PeriodicUpdater
}

func NewService(config *config.Config) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	client := NewCoinGeckoClient(config)

	service := &Service{
		config:              config,
		client:              client,
		metricsWriter:       metricsWriter,
		subscriptionManager: events.NewSubscriptionManager(),
	}

	service.periodicUpdater = NewPeriodicUpdater(
		config.TokenListFetcher,
		client,
		metricsWriter,
		service.onTokenListsUpdated,
	)

	return service
}

// onTokenListsUpdated is the callback called when token lists are updated
func (s *Service) onTokenListsUpdated(ctx context.Context, tokenLists map[string]*TokenList) error {
	now := time.Now().Unix()

	for platform, tokenList := range tokenLists {
		s.cache.Store(platform, &TokenListCache{
			Platform:  platform,
			TokenList: *tokenList,
			UpdatedAt: now,
		})
	}

	s.subscriptionManager.Emit(ctx)

	return nil
}

func (s *Service) Start(ctx context.Context) error {
	return s.periodicUpdater.Start(ctx)
}

func (s *Service) Stop() {
	s.periodicUpdater.Stop()
}

// GetTokenList returns cached token list for a specific platform
func (s *Service) GetTokenList(platform string) TokenListResponse {
	isSupported := false
	for _, supportedPlatform := range s.config.TokenListFetcher.SupportedPlatforms {
		if supportedPlatform == platform {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return TokenListResponse{
			TokenList: nil,
			Error:     fmt.Errorf("platform '%s' is not supported", platform),
		}
	}

	if value, exists := s.cache.Load(platform); exists {
		if tokenListCache, ok := value.(*TokenListCache); ok {
			tokenListCopy := tokenListCache.TokenList
			return TokenListResponse{
				TokenList: &tokenListCopy,
				Error:     nil,
			}
		}
	}

	return TokenListResponse{
		TokenList: nil,
		Error:     fmt.Errorf("token list for platform '%s' not available", platform),
	}
}

func (s *Service) Healthy() bool {
	empty := true
	s.cache.Range(func(_, _ interface{}) bool {
		empty = false
		return false
	})

	return s.periodicUpdater.IsInitialized() && !empty
}

func (s *Service) SubscribeOnTokenListsUpdate() events.ISubscription {
	return s.subscriptionManager.Subscribe()
}
