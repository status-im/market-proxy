package e2etest

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/status-im/market-proxy/core"
)

// TestEnv represents a test environment
type TestEnv struct {
	Registry      *core.Registry
	MockServer    *MockServer
	Context       context.Context
	CancelFunc    context.CancelFunc
	ConfigPath    string
	ServerBaseURL string
}

// SetupTest sets up the test environment
func SetupTest(t *testing.T) *TestEnv {
	// Create a context with cancellation capability
	ctx, cancel := context.WithCancel(context.Background())

	// Create a mock server
	mockServer := NewMockServer()

	// Load test configuration with URLs from the mock server
	cfg, configPath, err := loadTestConfig(mockServer.GetURL(), mockServer.GetWSURL())
	if err != nil {
		mockServer.Close()
		cancel()
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Use a specific port for testing
	testPort := "8081"
	os.Setenv("PORT", testPort)

	// Initialize services
	registry, err := core.Setup(ctx, cfg)
	if err != nil {
		cleanupTestConfig(configPath)
		mockServer.Close()
		cancel()
		t.Fatalf("Failed to setup services: %v", err)
	}

	// Start services
	if err := registry.StartAll(ctx); err != nil {
		registry.StopAll()
		cleanupTestConfig(configPath)
		mockServer.Close()
		cancel()
		t.Fatalf("Failed to start services: %v", err)
	}

	// Wait for the server to fully start
	time.Sleep(500 * time.Millisecond)

	// Determine the base API URL using the port from environment
	serverBaseURL := fmt.Sprintf("http://localhost:%s", testPort)

	// Add proxy rules from real server to mock server
	mockServer.AddProxyRulesToRealServer(serverBaseURL)

	// Check that the server is running and responding
	resp, err := http.Get(serverBaseURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		registry.StopAll()
		cleanupTestConfig(configPath)
		mockServer.Close()
		cancel()
		if err != nil {
			t.Fatalf("Server not responding: %v", err)
		} else {
			t.Fatalf("Server returned unexpected status: %d", resp.StatusCode)
		}
	}

	return &TestEnv{
		Registry:      registry,
		MockServer:    mockServer,
		Context:       ctx,
		CancelFunc:    cancel,
		ConfigPath:    configPath,
		ServerBaseURL: serverBaseURL,
	}
}

// TearDown releases test environment resources
func (env *TestEnv) TearDown() {
	if env.Registry != nil {
		env.Registry.StopAll()
	}
	if env.MockServer != nil {
		env.MockServer.Close()
	}
	if env.CancelFunc != nil {
		env.CancelFunc()
	}
	if env.ConfigPath != "" {
		cleanupTestConfig(env.ConfigPath)
	}
}
