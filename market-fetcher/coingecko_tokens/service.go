package coingecko_tokens

import (
	"context"
	"sync"

	"github.com/status-im/market-proxy/interfaces"

	"github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/metrics"
)

type Service struct {
	config              *config.Config
	client              *Client
	metricsWriter       *metrics.MetricsWriter
	subscriptionManager *events.SubscriptionManager
	cache               struct {
		sync.RWMutex
		tokens   []interfaces.Token
		tokenIds []string
	}
	periodicUpdater *PeriodicUpdater
}

func NewService(config *config.Config) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	baseURL := config.OverrideCoingeckoPublicURL
	if baseURL == "" {
		baseURL = coingecko_common.COINGECKO_PUBLIC_URL
	}

	client := NewClient(baseURL, metricsWriter)

	service := &Service{
		config:              config,
		client:              client,
		metricsWriter:       metricsWriter,
		subscriptionManager: events.NewSubscriptionManager(),
	}

	// Create periodic updater with callback
	service.periodicUpdater = NewPeriodicUpdater(
		config.TokensFetcher,
		client,
		metricsWriter,
		service.onTokensUpdated,
	)

	return service
}

// onTokensUpdated is the callback called when tokens are updated
func (s *Service) onTokensUpdated(ctx context.Context, tokens []interfaces.Token) error {
	// Extract token IDs
	tokenIds := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token.ID != "" {
			tokenIds = append(tokenIds, token.ID)
		}
	}

	// Update cache with both tokens and precomputed IDs
	s.cache.Lock()
	s.cache.tokens = tokens
	s.cache.tokenIds = tokenIds
	s.cache.Unlock()

	// Emit update notification
	s.subscriptionManager.Emit(ctx)

	return nil
}

func (s *Service) Start(ctx context.Context) error {
	return s.periodicUpdater.Start(ctx)
}

func (s *Service) Stop() {
	s.periodicUpdater.Stop()
}

// GetTokens returns cached tokens
func (s *Service) GetTokens() []interfaces.Token {
	s.cache.RLock()
	defer s.cache.RUnlock()

	// Return copy to avoid race conditions
	tokensCopy := make([]interfaces.Token, len(s.cache.tokens))
	copy(tokensCopy, s.cache.tokens)

	return tokensCopy
}

// GetTokenIds returns cached token IDs
func (s *Service) GetTokenIds() []string {
	s.cache.RLock()
	defer s.cache.RUnlock()

	// Return copy of precomputed token IDs to avoid race conditions
	tokenIdsCopy := make([]string, len(s.cache.tokenIds))
	copy(tokenIdsCopy, s.cache.tokenIds)

	return tokenIdsCopy
}

// Healthy checks if service is initialized and has data
func (s *Service) Healthy() bool {
	s.cache.RLock()
	tokensLen := len(s.cache.tokens)
	s.cache.RUnlock()

	return s.periodicUpdater.IsInitialized() && tokensLen > 0
}

func (s *Service) SubscribeOnTokensUpdate() events.SubscriptionInterface {
	return s.subscriptionManager.Subscribe()
}
