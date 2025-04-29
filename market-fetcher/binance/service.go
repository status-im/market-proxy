package binance

import (
	"context"
	"sync/atomic"

	"github.com/status-im/market-proxy/config"
)

type Service struct {
	config *config.Config
	// WebSocket client
	wsClient *WebSocketClient
	// Quotes manager
	quotes *QuotesManager
	// Flag indicating if at least one successful update was received
	successfulUpdate atomic.Bool
}

func NewService(cfg *config.Config) *Service {
	s := &Service{
		config: cfg,
		quotes: NewQuotesManager(),
	}

	// Create WebSocket client with message handler that also updates our health status
	s.wsClient = NewWebSocketClient(func(message []byte) error {
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
	// Connect to WebSocket with the provided context
	if err := s.wsClient.Connect(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Service) Stop() {
	s.wsClient.Close()
}
