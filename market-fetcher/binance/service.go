package binance

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/status-im/market-proxy/config"

	"github.com/gorilla/websocket"
)

const (
	BASE_WS_URL = "wss://data-stream.binance.vision/ws/!ticker@arr"
	// Connection timeouts
	PING_INTERVAL = 20 * time.Second
	PONG_TIMEOUT  = 60 * time.Second
)

type Service struct {
	config *config.Config
	// WebSocket connection
	conn *websocket.Conn
	// Channel to signal service stop
	stopCh chan struct{}
	// Quotes manager
	quotes *QuotesManager
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
		stopCh: make(chan struct{}),
		quotes: NewQuotesManager(),
	}
}

// SetWatchList sets the list of symbols to watch with the quote symbol
func (s *Service) SetWatchList(baseSymbols []string, quoteSymbol string) {
	s.quotes.SetWatchList(baseSymbols, quoteSymbol)
}

// GetLatestQuotes returns the latest quotes for watched symbols
func (s *Service) GetLatestQuotes() map[string]Quote {
	return s.quotes.GetLatestQuotes()
}

func (s *Service) connect() (*websocket.Conn, error) {
	// Connect to Binance WebSocket data stream
	conn, _, err := websocket.DefaultDialer.Dial(BASE_WS_URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Binance WebSocket: %v", err)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(PONG_TIMEOUT))

	return conn, nil
}

func (s *Service) handlePingPong(conn *websocket.Conn) {
	// Start ping handler
	go func() {
		ticker := time.NewTicker(PING_INTERVAL)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				// Send empty pong frame
				if err := conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(PING_INTERVAL)); err != nil {
					log.Printf("Error sending pong: %v", err)
					return
				}
			}
		}
	}()

	// Set ping handler
	conn.SetPingHandler(func(string) error {
		// Reset read deadline
		conn.SetReadDeadline(time.Now().Add(PONG_TIMEOUT))
		return nil
	})
}

func (s *Service) Start(ctx context.Context) error {
	// Create WebSocket connection
	conn, err := s.connect()
	if err != nil {
		return err
	}
	s.conn = conn

	// Setup ping/pong handling
	s.handlePingPong(conn)

	// Start message handler
	go s.handleMessages(ctx)

	return nil
}

func (s *Service) handleMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		default:
			// Read message
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Error reading WebSocket message: %v", err)
				}
				// Try to reconnect
				s.reconnect(ctx)
				continue
			}

			// Update quotes
			if err := s.quotes.UpdateQuotes(message); err != nil {
				log.Printf("%v", err)
			}
		}
	}
}

func (s *Service) reconnect(ctx context.Context) {
	if s.conn != nil {
		s.conn.Close()
	}

	conn, err := s.connect()
	if err != nil {
		log.Printf("Failed to reconnect: %v", err)
		return
	}

	s.conn = conn
	s.handlePingPong(conn)
}

func (s *Service) Stop() {
	close(s.stopCh)
	if s.conn != nil {
		s.conn.Close()
	}
}
