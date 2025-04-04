package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/status-im/market-proxy/api"
	"github.com/status-im/market-proxy/binance"
	"github.com/status-im/market-proxy/coingecko"
	"github.com/status-im/market-proxy/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Load CoinGecko API tokens
	cgApiTokens, err := config.LoadAPITokens(cfg.CoinGeckoFetcher.TokensFile)
	if err != nil {
		log.Fatal("Error loading CoinGecko API tokens:", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping services...")
		cancel()
	}()

	// Create Binance service first
	binanceService := binance.NewService(cfg)

	// Start the Binance service
	if err := binanceService.Start(ctx); err != nil {
		log.Fatal("Failed to start Binance service:", err)
	}
	defer binanceService.Stop()

	// Create channel for updates
	updateCh := make(chan struct{}, 1)

	// Create CoinGecko service with callback
	cgService := coingecko.NewService(cfg, cgApiTokens, func() {
		select {
		case updateCh <- struct{}{}:
		default:
			// Channel is full, skip sending
		}
	})

	// Start the CoinGecko service
	if err := cgService.Start(ctx); err != nil {
		log.Fatal("Failed to start CoinGecko service:", err)
	}
	defer cgService.Stop()

	// Handle updates from CoinMarketCap and CoinGecko
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-updateCh:
				// Get latest data from CoinGecko cache
				cgData := cgService.GetCacheData()
				if cgData != nil {
					// Extract symbols
					symbols := make([]string, 0, len(cgData.Data))
					for _, coin := range cgData.Data {
						// Convert symbols to uppercase as Binance API requires
						symbols = append(symbols, strings.ToUpper(coin.Symbol))
					}

					// Update Binance watchlist with a specific name
					binanceService.SetWatchList(symbols, "USDT")
				}
			}
		}
	}()

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create and start HTTP server
	server := api.New(port, binanceService, cgService)
	if err := server.Start(); err != nil {
		log.Fatal("Server failed:", err)
	}
}
