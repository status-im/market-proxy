package coingecko_tokens

import (
	"context"
	"log"
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
	tokenIds := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token.ID != "" {
			tokenIds = append(tokenIds, token.ID)
		}
	}

	s.cache.Lock()
	s.cache.tokens = tokens
	s.cache.tokenIds = tokenIds
	tokensCount := len(s.cache.tokens)
	tokenIdsCount := len(s.cache.tokenIds)
	s.cache.Unlock()

	// Log cache statistics for tokens service
	log.Printf("Tokens service cache update complete - cached tokens: %d, cached token IDs: %d", tokensCount, tokenIdsCount)

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

	tokensCopy := make([]interfaces.Token, len(s.cache.tokens))
	copy(tokensCopy, s.cache.tokens)

	return tokensCopy
}

// GetTokenIds returns cached token IDs
func (s *Service) GetTokenIds() []string {
	s.cache.RLock()
	defer s.cache.RUnlock()

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

func (s *Service) SubscribeOnTokensUpdate() events.ISubscription {
	return s.subscriptionManager.Subscribe()
}
