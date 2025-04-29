package binance

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/status-im/market-proxy/config"
)

type Service struct {
	config *config.Config
	// WebSocket client
	wsClient *WebSocketClient
	// Channel to signal service stop
	stopCh chan struct{}
	// Quotes manager
	quotes *QuotesManager
	// Flag indicating if at least one successful update was received
	successfulUpdate atomic.Bool
}

func NewService(cfg *config.Config) *Service {
	s := &Service{
		config: cfg,
		stopCh: make(chan struct{}),
		quotes: NewQuotesManager(),
	}

	// Create WebSocket client with message handler that also updates our health status
	s.wsClient = NewWebSocketClient(s.stopCh, func(message []byte) error {
		err := s.quotes.UpdateQuotes(message)
		if err == nil {
			// Mark that we've received at least one successful update
			s.successfulUpdate.Store(true)
		}
		return err
	}, cfg.OverrideBinanceWSURL)

	return s
}

// SetWatchList sets the list of symbols to watch with the quote symbol
func (s *Service) SetWatchList(baseSymbols []string, quoteSymbol string) {
	s.quotes.SetWatchList(baseSymbols, quoteSymbol)
}

// GetLatestQuotes returns the latest quotes for watched symbols
func (s *Service) GetLatestQuotes() map[string]Quote {
	return s.quotes.GetLatestQuotes()
}

// Healthy returns true if at least one successful update has been received
func (s *Service) Healthy() bool {
	return s.successfulUpdate.Load()
}

func (s *Service) Start(ctx context.Context) error {
	// Create WebSocket connection
	if err := s.wsClient.Connect(); err != nil {
		return err
	}

	// Setup ping/pong handling
	s.wsClient.SetupPingPong()

	// Start message handler
	go s.wsClient.StartMessageLoop(ctx, func() {
		s.reconnect(ctx)
	})

	return nil
}

func (s *Service) reconnect(ctx context.Context) {
	s.wsClient.Close()

	if err := s.wsClient.Connect(); err != nil {
		log.Printf("Failed to reconnect: %v", err)
		return
	}

	s.wsClient.SetupPingPong()
}

func (s *Service) Stop() {
	close(s.stopCh)
	s.wsClient.Close()
}
