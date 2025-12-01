package fetcher_by_id

import (
	"context"
	"log"
	"time"

	"github.com/status-im/market-proxy/coingecko_common"
)

const (
	ChunksDefaultChunkSize    = 250
	ChunksDefaultRequestDelay = 1000
)

// ChunksFetcher handles fetching data in chunks
type ChunksFetcher struct {
	client       *Client
	name         string
	chunkSize    int
	requestDelay time.Duration
	isBatchMode  bool
}

func NewChunksFetcher(client *Client, name string, chunkSize int, requestDelayMs int, isBatchMode bool) *ChunksFetcher {
	var requestDelay time.Duration
	if requestDelayMs >= 0 {
		requestDelay = time.Duration(requestDelayMs) * time.Millisecond
	} else {
		requestDelay = ChunksDefaultRequestDelay * time.Millisecond
	}

	if chunkSize <= 0 {
		chunkSize = ChunksDefaultChunkSize
	}

	return &ChunksFetcher{
		client:       client,
		name:         name,
		chunkSize:    chunkSize,
		requestDelay: requestDelay,
		isBatchMode:  isBatchMode,
	}
}

func (f *ChunksFetcher) FetchData(ctx context.Context, ids []string, onChunk func(map[string][]byte)) (map[string][]byte, error) {
	if len(ids) == 0 {
		return make(map[string][]byte), nil
	}

	if f.isBatchMode {
		return f.fetchBatch(ctx, ids, onChunk)
	}
	return f.fetchSingle(ctx, ids, onChunk)
}

// fetchSingle fetches data for each ID individually
func (f *ChunksFetcher) fetchSingle(ctx context.Context, ids []string, onChunk func(map[string][]byte)) (map[string][]byte, error) {
	startTime := time.Now()
	log.Printf("%s: Fetching data for %d IDs in single mode", f.name, len(ids))

	chunkSize := 1

	fetchFunc := func(ctx context.Context, chunk []string) (map[string][]byte, error) {
		if len(chunk) == 0 {
			return make(map[string][]byte), nil
		}

		id := chunk[0]
		result, err := f.client.FetchSingle(id)
		if err != nil {
			log.Printf("%s: Failed to fetch %s: %v", f.name, id, err)
			return nil, err
		}

		chunkData := map[string][]byte{id: result}

		if onChunk != nil {
			onChunk(chunkData)
		}

		return chunkData, nil
	}

	result, err := coingecko_common.ChunkMapFetcher(
		ctx,
		ids,
		chunkSize,
		coingecko_common.MaxChunkStringLength,
		f.requestDelay,
		fetchFunc,
	)
	if err != nil {
		return nil, err
	}

	tokensPerSecond := float64(len(ids)) / time.Since(startTime).Seconds()
	log.Printf("%s: Single-mode fetch complete: %d items (%.2f items/sec)",
		f.name, len(result), tokensPerSecond)

	return result, nil
}

func (f *ChunksFetcher) fetchBatch(ctx context.Context, ids []string, onChunk func(map[string][]byte)) (map[string][]byte, error) {
	startTime := time.Now()
	numChunks := (len(ids) + f.chunkSize - 1) / f.chunkSize
	log.Printf("%s: Fetching data for %d IDs in %d chunks (batch mode)", f.name, len(ids), numChunks)

	fetchFunc := func(ctx context.Context, chunk []string) (map[string][]byte, error) {
		chunkData, err := f.client.FetchBatch(chunk)
		if err != nil {
			log.Printf("%s: Error fetching chunk: %v", f.name, err)
			return nil, err
		}

		if onChunk != nil {
			onChunk(chunkData)
		}

		return chunkData, nil
	}

	result, err := coingecko_common.ChunkMapFetcher(
		ctx,
		ids,
		f.chunkSize,
		coingecko_common.MaxChunkStringLength,
		f.requestDelay,
		fetchFunc,
	)
	if err != nil {
		return nil, err
	}

	tokensPerSecond := float64(len(ids)) / time.Since(startTime).Seconds()
	log.Printf("%s: Batch-mode fetch complete: %d items (%.2f items/sec, %d chunks)",
		f.name, len(result), tokensPerSecond, numChunks)

	return result, nil
}
