package coingecko_tokens

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/status-im/market-proxy/interfaces"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

// UpdatedCallback is called when tokens are successfully updated
type UpdatedCallback func(ctx context.Context, tokens []interfaces.Token) error

// PeriodicUpdater handles periodic fetching and updating of tokens
type PeriodicUpdater struct {
	config        config.CoingeckoCoinslistFetcher
	client        *Client
	metricsWriter *metrics.MetricsWriter
	onUpdated     UpdatedCallback
	scheduler     *scheduler.Scheduler
	initialized   atomic.Bool
}

// NewPeriodicUpdater creates a new periodic updater
func NewPeriodicUpdater(
	config config.CoingeckoCoinslistFetcher,
	client *Client,
	metricsWriter *metrics.MetricsWriter,
	onUpdated UpdatedCallback,
) *PeriodicUpdater {
	return &PeriodicUpdater{
		config:        config,
		client:        client,
		metricsWriter: metricsWriter,
		onUpdated:     onUpdated,
	}
}

// Start begins periodic updates
func (u *PeriodicUpdater) Start(ctx context.Context) error {
	updateInterval := u.config.UpdateInterval

	// Skip periodic updates if interval is 0 or negative
	if updateInterval <= 0 {
		log.Printf("Tokens periodic updater: periodic updates disabled (interval: %v)", updateInterval)
		return nil
	}

	u.scheduler = scheduler.New(updateInterval, func(ctx context.Context) {
		if err := u.fetchAndUpdate(ctx); err != nil {
			log.Printf("Error updating tokens: %v", err)
		} else {
			u.initialized.Store(true)
		}
	})

	u.scheduler.Start(ctx, true)

	return nil
}

// Stop stops periodic updates
func (u *PeriodicUpdater) Stop() {
	if u.scheduler != nil {
		u.scheduler.Stop()
	}
}

// IsInitialized returns true if updater has successfully fetched data at least once
func (u *PeriodicUpdater) IsInitialized() bool {
	return u.initialized.Load()
}

// fetchAndUpdate fetches tokens from API and calls the callback
func (u *PeriodicUpdater) fetchAndUpdate(ctx context.Context) error {
	u.metricsWriter.ResetCycleMetrics()
	startTime := time.Now()

	tokens, err := u.client.FetchTokens()
	if err != nil {
		return fmt.Errorf("failed to fetch tokens: %w", err)
	}

	filteredTokens := FilterTokensByPlatform(tokens, u.config.SupportedPlatforms)

	tokensByPlatform := CountTokensByPlatform(filteredTokens)

	u.metricsWriter.RecordDataFetchCycle(time.Since(startTime))
	u.metricsWriter.RecordCacheSize(len(filteredTokens))
	metrics.RecordTokensByPlatform(tokensByPlatform)

	// Call the callback with updated tokens
	if u.onUpdated != nil {
		if err := u.onUpdated(ctx, filteredTokens); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	log.Printf("Updated tokens cache, now contains %d tokens with supported platforms", len(filteredTokens))
	return nil
}
