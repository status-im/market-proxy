package coingecko_exchange_rates

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/metrics"
	"github.com/status-im/market-proxy/scheduler"
)

// Service periodically fetches CoinGecko /api/v3/exchange_rates and serves the
// last successful body verbatim (Passthrough).
//
// It is intentionally independent of the currency ratios service: conversion
// Estimates never depend on this endpoint.
type Service struct {
	config        config.ExchangeRatesFetcherConfig
	client        IClient
	metricsWriter *metrics.MetricsWriter
	scheduler     *scheduler.Scheduler
	cache         atomic.Pointer[ExchangeRatesResponse]
}

// NewService creates an exchange rates service
func NewService(cfg *config.Config) *Service {
	metricsWriter := metrics.NewMetricsWriter(metrics.ServiceExchangeRates)
	return NewServiceWithClient(cfg.CoingeckoExchangeRates, NewCoinGeckoClient(cfg, metricsWriter), metricsWriter)
}

// NewServiceWithClient creates an exchange rates service with an explicit client
func NewServiceWithClient(cfg config.ExchangeRatesFetcherConfig, client IClient, metricsWriter *metrics.MetricsWriter) *Service {
	return &Service{
		config:        cfg,
		client:        client,
		metricsWriter: metricsWriter,
	}
}

// Start begins periodic updates
func (s *Service) Start(ctx context.Context) error {
	if s.config.UpdateInterval <= 0 {
		log.Printf("Exchange rates service: periodic updates disabled (interval: %v)", s.config.UpdateInterval)
		return nil
	}

	s.scheduler = scheduler.New(s.config.UpdateInterval, func(ctx context.Context) {
		if err := s.fetchAndUpdate(); err != nil {
			log.Printf("Exchange rates service: update failed, keeping previous snapshot: %v", err)
		}
	})

	s.scheduler.Start(ctx, true)

	return nil
}

// Stop terminates periodic updates
func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

// ExchangeRates returns the cached exchange rates body, or an error if none was fetched yet
func (s *Service) ExchangeRates() (ExchangeRatesResponse, error) {
	cached := s.cache.Load()
	if cached == nil {
		return nil, fmt.Errorf("exchange rates are not available yet")
	}
	return *cached, nil
}

// Healthy reports whether exchange rates data is available
func (s *Service) Healthy() bool {
	return s.cache.Load() != nil
}

func (s *Service) fetchAndUpdate() error {
	s.metricsWriter.ResetCycleMetrics()
	defer s.metricsWriter.TrackDataFetchCycle()()

	rates, err := s.client.FetchExchangeRates()
	if err != nil {
		return err
	}

	s.cache.Store(&rates)
	s.metricsWriter.RecordCacheSize(1)

	return nil
}
