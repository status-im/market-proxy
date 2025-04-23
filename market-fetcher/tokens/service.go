package tokens

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/scheduler"
)

// Service represents the tokens service that periodically fetches and filters token data
type Service struct {
	config   *config.Config
	client   *Client
	onUpdate func()
	cache    struct {
		sync.RWMutex
		tokens []Token
	}
	scheduler   *scheduler.Scheduler
	initialized bool
}

// NewService creates a new tokens service
func NewService(config *config.Config, onUpdate func()) *Service {
	client := NewClient(DefaultCoinGeckoBaseURL)

	return &Service{
		config:   config,
		client:   client,
		onUpdate: onUpdate,
	}
}

// Start starts the tokens service
func (s *Service) Start(ctx context.Context) error {
	// Start periodic updates
	updateInterval := time.Duration(s.config.TokensFetcher.UpdateIntervalMs) * time.Millisecond

	// Create and start the scheduler
	s.scheduler = scheduler.New(updateInterval, func(ctx context.Context) {
		if err := s.fetchAndUpdate(); err != nil {
			log.Printf("Error updating tokens: %v", err)
		} else {
			s.initialized = true
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
	tokens, err := s.client.FetchTokens()
	if err != nil {
		return fmt.Errorf("failed to fetch tokens: %w", err)
	}

	// Filter tokens by keeping only supported platforms
	filteredTokens := FilterTokensByPlatform(tokens, s.config.TokensFetcher.SupportedPlatforms)

	s.cache.Lock()
	s.cache.tokens = filteredTokens
	s.cache.Unlock()

	// Signal update through callback if provided
	if s.onUpdate != nil {
		s.onUpdate()
	}

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
	defer s.cache.RUnlock()

	return s.initialized && len(s.cache.tokens) > 0
}
