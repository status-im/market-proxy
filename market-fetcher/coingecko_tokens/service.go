package coingecko_tokens

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

// Service represents the tokens service that periodically fetches and filters token data
type Service struct {
	config        *config.Config
	client        *Client
	metricsWriter *metrics.MetricsWriter
	cache         struct {
		sync.RWMutex
		tokens []Token
	}
	scheduler   *scheduler.Scheduler
	initialized atomic.Bool
}

// NewService creates a new tokens service
func NewService(config *config.Config) *Service {
	// Create metrics writer
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCoins)

	baseURL := config.OverrideCoingeckoPublicURL
	if baseURL == "" {
		baseURL = coingecko_common.COINGECKO_PUBLIC_URL
	}

	client := NewClient(baseURL, metricsWriter)

	return &Service{
		config:        config,
		client:        client,
		metricsWriter: metricsWriter,
	}
}

// Start starts the tokens service
func (s *Service) Start(ctx context.Context) error {
	// Start periodic updates - use update interval directly as it's already a time.Duration
	updateInterval := s.config.TokensFetcher.UpdateInterval

	// Create and start the scheduler
	s.scheduler = scheduler.New(updateInterval, func(ctx context.Context) {
		if err := s.fetchAndUpdate(); err != nil {
			log.Printf("Error updating tokens: %v", err)
		} else {
			s.initialized.Store(true)
		}
	})

	// The scheduler will execute the task immediately on start
	s.scheduler.Start(ctx, true)

	return nil
}

// Stop stops the tokens service
func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

// fetchAndUpdate fetches data from CoinGecko and updates the cache
func (s *Service) fetchAndUpdate() error {
	// Reset request cycle counters
	s.metricsWriter.ResetCycleMetrics()

	// Record start time for metrics
	startTime := time.Now()

	tokens, err := s.client.FetchTokens()
	if err != nil {
		return fmt.Errorf("failed to fetch tokens: %w", err)
	}

	// Filter tokens by keeping only supported platforms
	filteredTokens := FilterTokensByPlatform(tokens, s.config.TokensFetcher.SupportedPlatforms)

	s.cache.Lock()
	s.cache.tokens = filteredTokens
	s.cache.Unlock()

	// Count tokens per platform
	tokensByPlatform := CountTokensByPlatform(filteredTokens)

	// Record metrics using MetricsWriter
	s.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
	s.metricsWriter.RecordCacheSize(len(filteredTokens))
	// Record tokens by platform - this still uses the global metric as it's platform-specific
	metrics.RecordTokensByPlatform(tokensByPlatform)

	log.Printf("Updated tokens cache, now contains %d tokens with supported platforms", len(filteredTokens))
	return nil
}

// GetTokens returns the cached tokens
func (s *Service) GetTokens() []Token {
	s.cache.RLock()
	defer s.cache.RUnlock()

	// Return a copy to avoid race conditions
	tokensCopy := make([]Token, len(s.cache.tokens))
	copy(tokensCopy, s.cache.tokens)

	return tokensCopy
}

// Healthy checks if the service is initialized and has data
func (s *Service) Healthy() bool {
	s.cache.RLock()
	tokensLen := len(s.cache.tokens)
	s.cache.RUnlock()

	return s.initialized.Load() && tokensLen > 0
}
