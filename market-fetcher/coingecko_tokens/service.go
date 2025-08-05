package coingecko_tokens

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/status-im/market-proxy/interfaces"

	"github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

type Service struct {
	config              *config.Config
	client              *Client
	metricsWriter       *metrics.MetricsWriter
	subscriptionManager *events.SubscriptionManager
	cache               struct {
		sync.RWMutex
		tokens []interfaces.Token
	}
	scheduler   *scheduler.Scheduler
	initialized atomic.Bool
}

func NewService(config *config.Config) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	baseURL := config.OverrideCoingeckoPublicURL
	if baseURL == "" {
		baseURL = coingecko_common.COINGECKO_PUBLIC_URL
	}

	client := NewClient(baseURL, metricsWriter)

	return &Service{
		config:              config,
		client:              client,
		metricsWriter:       metricsWriter,
		subscriptionManager: events.NewSubscriptionManager(),
	}
}

func (s *Service) Start(ctx context.Context) error {
	updateInterval := s.config.TokensFetcher.UpdateInterval

	// Skip periodic updates if interval is 0 or negative
	if updateInterval <= 0 {
		log.Printf("Tokens service: periodic updates disabled (interval: %v)", updateInterval)
		return nil
	}

	s.scheduler = scheduler.New(updateInterval, func(ctx context.Context) {
		if err := s.fetchAndUpdate(ctx); err != nil {
			log.Printf("Error updating tokens: %v", err)
		} else {
			s.initialized.Store(true)
		}
	})

	s.scheduler.Start(ctx, true)

	return nil
}

func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

func (s *Service) fetchAndUpdate(ctx context.Context) error {
	s.metricsWriter.ResetCycleMetrics()
	startTime := time.Now()

	tokens, err := s.client.FetchTokens()
	if err != nil {
		return fmt.Errorf("failed to fetch tokens: %w", err)
	}

	filteredTokens := FilterTokensByPlatform(tokens, s.config.TokensFetcher.SupportedPlatforms)

	s.cache.Lock()
	s.cache.tokens = filteredTokens
	s.cache.Unlock()

	tokensByPlatform := CountTokensByPlatform(filteredTokens)

	s.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
	s.metricsWriter.RecordCacheSize(len(filteredTokens))
	metrics.RecordTokensByPlatform(tokensByPlatform)

	s.subscriptionManager.Emit(ctx)

	log.Printf("Updated tokens cache, now contains %d tokens with supported platforms", len(filteredTokens))
	return nil
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

// Healthy checks if service is initialized and has data
func (s *Service) Healthy() bool {
	s.cache.RLock()
	tokensLen := len(s.cache.tokens)
	s.cache.RUnlock()

	return s.initialized.Load() && tokensLen > 0
}

func (s *Service) SubscribeOnTokensUpdate() chan struct{} {
	return s.subscriptionManager.Subscribe()
}

func (s *Service) Unsubscribe(ch chan struct{}) {
	s.subscriptionManager.Unsubscribe(ch)
}
