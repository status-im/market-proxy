package fetcher_by_id

import (
	"context"
	"testing"
	"time"

	"github.com/status-im/market-proxy/config"
)

func TestChunksFetcher_BatchMode(t *testing.T) {
	fetcherCfg := &config.FetcherByIdConfig{
		Name:         "test",
		EndpointPath: "/api/v3/simple/price?ids={{ids_list}}",
		ChunkSize:    2,
	}

	fetcher := NewChunksFetcher(nil, fetcherCfg.Name, fetcherCfg.GetChunkSize(), 0, fetcherCfg.IsBatchMode())

	if !fetcher.isBatchMode {
		t.Error("Expected batch mode to be true")
	}

	if fetcher.chunkSize != 2 {
		t.Errorf("Expected chunk size 2, got %d", fetcher.chunkSize)
	}

	if fetcher.name != "test" {
		t.Errorf("Expected name 'test', got '%s'", fetcher.name)
	}
}

func TestChunksFetcher_SingleMode(t *testing.T) {
	fetcherCfg := &config.FetcherByIdConfig{
		Name:         "test",
		EndpointPath: "/api/v3/coins/{{id}}",
	}

	fetcher := NewChunksFetcher(nil, fetcherCfg.Name, fetcherCfg.GetChunkSize(), 0, fetcherCfg.IsBatchMode())

	if fetcher.isBatchMode {
		t.Error("Expected batch mode to be false")
	}
}

func TestChunksFetcher_DefaultChunkSize(t *testing.T) {
	// Test with zero chunk size (should use default)
	fetcher := NewChunksFetcher(nil, "test", 0, 100, false)

	if fetcher.chunkSize != ChunksDefaultChunkSize {
		t.Errorf("Expected default chunk size %d, got %d", ChunksDefaultChunkSize, fetcher.chunkSize)
	}
}

func TestChunksFetcher_DefaultRequestDelay(t *testing.T) {
	// Test with negative delay (should use default)
	fetcher := NewChunksFetcher(nil, "test", 100, -1, false)

	expectedDelay := time.Duration(ChunksDefaultRequestDelay) * time.Millisecond
	if fetcher.requestDelay != expectedDelay {
		t.Errorf("Expected default delay %v, got %v", expectedDelay, fetcher.requestDelay)
	}
}

func TestChunksFetcher_CustomRequestDelay(t *testing.T) {
	// Test with custom delay
	fetcher := NewChunksFetcher(nil, "test", 100, 500, false)

	expectedDelay := 500 * time.Millisecond
	if fetcher.requestDelay != expectedDelay {
		t.Errorf("Expected delay %v, got %v", expectedDelay, fetcher.requestDelay)
	}
}

func TestChunksFetcher_ZeroRequestDelay(t *testing.T) {
	// Test with zero delay (should work as no delay)
	fetcher := NewChunksFetcher(nil, "test", 100, 0, false)

	if fetcher.requestDelay != 0 {
		t.Errorf("Expected delay 0, got %v", fetcher.requestDelay)
	}
}

func TestChunksFetcher_FetchData_EmptyIDs(t *testing.T) {
	fetcher := NewChunksFetcher(nil, "test", 100, 0, false)

	result, err := fetcher.FetchData(context.Background(), []string{}, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}
