package coingecko_token_list

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

type UpdatedCallback func(ctx context.Context, tokenLists map[string]*TokenList) error

// PeriodicUpdater handles periodic fetching and updating of token lists
type PeriodicUpdater struct {
	config        config.TokenListFetcherConfig
	client        IClient
	metricsWriter *metrics.MetricsWriter
	onUpdated     UpdatedCallback
	scheduler     *scheduler.Scheduler
	initialized   atomic.Bool
}

func NewPeriodicUpdater(
	config config.TokenListFetcherConfig,
	client IClient,
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
		log.Printf("Token lists periodic updater: periodic updates disabled (interval: %v)", updateInterval)
		return nil
	}

	u.scheduler = scheduler.New(updateInterval, func(ctx context.Context) {
		if err := u.fetchAndUpdate(ctx); err != nil {
			log.Printf("Error updating token lists: %v", err)
		} else {
			u.initialized.Store(true)
		}
	})

	u.scheduler.Start(ctx, true)

	return nil
}

func (u *PeriodicUpdater) Stop() {
	if u.scheduler != nil {
		u.scheduler.Stop()
	}
}

func (u *PeriodicUpdater) IsInitialized() bool {
	return u.initialized.Load()
}

// fetchAndUpdate fetches token lists from API and calls the callback
func (u *PeriodicUpdater) fetchAndUpdate(ctx context.Context) error {
	u.metricsWriter.ResetCycleMetrics()
	defer u.metricsWriter.TrackDataFetchCycle()()

	tokenLists := make(map[string]*TokenList)
	var totalTokens int

	for _, platform := range u.config.SupportedPlatforms {
		tokenList, err := u.client.FetchTokenList(platform)
		if err != nil {
			log.Printf("Failed to fetch token list for platform %s: %v", platform, err)
			continue
		}

		tokenLists[platform] = tokenList
		totalTokens += len(tokenList.Tokens)
	}

	if len(tokenLists) == 0 {
		return fmt.Errorf("failed to fetch any token lists")
	}

	u.metricsWriter.RecordCacheSize(totalTokens)

	if u.onUpdated != nil {
		if err := u.onUpdated(ctx, tokenLists); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	return nil
}
