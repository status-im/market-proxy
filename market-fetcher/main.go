package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/core"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup services
	registry, err := core.Setup(ctx, cfg)
	if err != nil {
		log.Fatal("Failed to setup services:", err)
	}
	defer registry.StopAll()

	// Start all services
	if err := registry.StartAll(ctx); err != nil {
		log.Fatal("Failed to start services:", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal, stopping services...")
	cancel() // Cancel context to stop all services
}
