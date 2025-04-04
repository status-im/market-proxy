package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/scheduler"
)

const (
	// Base URL for public API
	COINGECKO_PUBLIC_URL = "https://api.coingecko.com/api/v3/coins/markets"
	// Base URL for Pro API
	COINGECKO_PRO_URL = "https://pro-api.coingecko.com/api/v3/coins/markets"
	// Maximum items per page
	MAX_PER_PAGE = 250 // CoinGecko's API max per_page value
)

type CacheData struct {
	sync.RWMutex
	Data interface{}
}

// Service represents the CoinGecko service
type Service struct {
	config    *config.Config
	apiTokens *config.APITokens
	onUpdate  func()
	cache     struct {
		sync.RWMutex
		data *APIResponse
	}
	scheduler *scheduler.Scheduler
}

// NewService creates a new CoinGecko service
func NewService(cfg *config.Config, apiTokens *config.APITokens, onUpdate func()) *Service {
	return &Service{
		config:    cfg,
		apiTokens: apiTokens,
		onUpdate:  onUpdate,
	}
}

// Returns the appropriate API URL based on whether we have an API key
func (s *Service) getApiBaseUrl() string {
	if len(s.apiTokens.Tokens) > 0 {
		if s.isUsingDemoKey() {
			log.Printf("CoinGecko: Detected Demo API key, using public API URL")
			return COINGECKO_PUBLIC_URL
		} else {
			log.Printf("CoinGecko: Using Pro API URL with API key")
			return COINGECKO_PRO_URL
		}
	}
	return COINGECKO_PUBLIC_URL
}

// Determines if the API key is a demo key
func (s *Service) isUsingDemoKey() bool {
	if len(s.apiTokens.Tokens) == 0 {
		return false
	}

	apiKey := s.apiTokens.Tokens[0]
	// Check if this is a demo key
	return strings.HasPrefix(apiKey, "demo_") ||
		strings.HasPrefix(apiKey, "CG-") ||
		strings.Contains(strings.ToLower(apiKey), "demo")
}

// Start starts the CoinGecko service
func (s *Service) Start(ctx context.Context) error {
	// Create scheduler for periodic updates
	s.scheduler = scheduler.New(
		time.Duration(s.config.CoinGeckoFetcher.UpdateInterval)*time.Second,
		func(ctx context.Context) {
			if err := s.fetchAndUpdate(ctx); err != nil {
				log.Printf("Error updating data: %v", err)
			}
		},
	)

	// Start the initial fetch
	if err := s.fetchAndUpdate(ctx); err != nil {
		return fmt.Errorf("initial fetch failed: %v", err)
	}

	// Start the scheduler with context
	s.scheduler.Start(ctx)
	return nil
}

func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}

// fetchData fetches data from CoinGecko API
func (s *Service) fetchData() (*APIResponse, error) {
	client := &http.Client{}

	// Create a random number generator with current time seed
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Calculate how many pages we need
	totalLimit := s.config.CoinGeckoFetcher.Limit
	perPage := MAX_PER_PAGE // CoinGecko limits to 250 per page

	if totalLimit <= perPage {
		// If limit is less than max per page, just fetch one page
		log.Printf("CoinGecko: Small request, fetching single page with %d coins", totalLimit)
		return s.fetchSinglePage(client, 1, totalLimit)
	}

	// Need multiple pages
	totalPages := (totalLimit + perPage - 1) / perPage // Ceiling division
	log.Printf("CoinGecko: Fetching %d coins from CoinGecko in %d pages using sequential requests", totalLimit, totalPages)

	// Use a rate limiter to control API request speed
	// Start with 1 request per 1.5 seconds (conservative)
	requestInterval := 1500 * time.Millisecond
	// For Pro API subscribers, we can be more aggressive
	if len(s.apiTokens.Tokens) > 0 {
		// With API key we can make about 30 requests per minute = 2 seconds per request
		// Being conservative with 2.5s between requests to avoid rate limits
		requestInterval = 2500 * time.Millisecond
	}

	// Track metrics
	startTime := time.Now()
	completedPages := 0
	retriedPages := 0
	var allCoins []CoinData

	// Fetch pages sequentially with rate limiting
	for page := 1; page <= totalPages; page++ {
		// Calculate limit for this page
		pageLimit := perPage
		if page == totalPages {
			pageLimit = totalLimit - (page-1)*perPage
		}

		if page > 1 {
			// Add jitter to the delay
			jitter := time.Duration(r.Intn(500)) * time.Millisecond // 0-500ms jitter
			delay := requestInterval + jitter
			log.Printf("CoinGecko: Rate limiting - waiting %.2fs before fetching page %d", delay.Seconds(), page)
			time.Sleep(delay)
		}

		pageStartTime := time.Now()
		log.Printf("CoinGecko: Starting fetch for page %d with limit %d", page, pageLimit)
		pageData, err := s.fetchSinglePage(client, page, pageLimit)
		if err != nil {
			// If this was a rate limit error and we've only completed a few pages,
			// increase the interval for subsequent requests
			if strings.Contains(err.Error(), "rate limit exceeded") && completedPages < 3 {
				oldInterval := requestInterval
				// Double the interval
				requestInterval = requestInterval * 2
				log.Printf("CoinGecko: Rate limit hit early, increasing request interval from %.1fs to %.1fs",
					oldInterval.Seconds(), requestInterval.Seconds())

				// If we hit rate limits very early, it might be better to fetch fewer pages
				if completedPages < 2 && totalPages > 5 {
					newTotal := totalPages / 2
					if newTotal < 3 {
						newTotal = 3
					}
					log.Printf("CoinGecko: Rate limit hit very early, reducing target from %d to %d pages",
						totalPages, newTotal)
					totalPages = newTotal
					totalLimit = newTotal * perPage
				}

				// Wait longer before retrying
				extraWait := 5 * time.Second
				log.Printf("CoinGecko: Waiting an extra %.1fs before continuing", extraWait.Seconds())
				time.Sleep(extraWait)

				// Retry this page after adapting
				page--
				retriedPages++
				continue
			}

			// For other errors or if we've tried to adapt multiple times, log and continue
			if retriedPages > 5 {
				log.Printf("CoinGecko: Too many retries, continuing with partial data")
			} else {
				log.Printf("CoinGecko: Error fetching page %d: %v", page, err)
			}

			// If we have some data already, continue with what we have
			if len(allCoins) > 0 {
				log.Printf("CoinGecko: Continuing with partial data (%d coins so far)", len(allCoins))
				break
			}

			// If we have no data at all, propagate the error
			return nil, fmt.Errorf("failed to fetch initial data: %v", err)
		}

		// Add coins from this page to the result
		allCoins = append(allCoins, pageData.Data...)

		// Update metrics
		completedPages++
		progress := float64(completedPages) / float64(totalPages) * 100
		pageTime := time.Since(pageStartTime)
		log.Printf("CoinGecko: Completed page %d/%d (%.1f%%) with %d coins in %.2fs",
			page, totalPages, progress, len(pageData.Data), pageTime.Seconds())

		// If we've got enough data, stop
		if len(allCoins) >= totalLimit {
			log.Printf("CoinGecko: Reached target limit of %d coins after %d pages", totalLimit, completedPages)
			break
		}
	}

	// Trim to requested limit if we got more
	if len(allCoins) > totalLimit {
		allCoins = allCoins[:totalLimit]
	}

	totalTime := time.Since(startTime)
	log.Printf("CoinGecko: Successfully fetched %d coins in %.2fs (%.2f coins/sec)",
		len(allCoins), totalTime.Seconds(), float64(len(allCoins))/totalTime.Seconds())

	// If we didn't get all the data we wanted, but have some, return what we have with a warning
	if len(allCoins) < totalLimit && len(allCoins) > 0 {
		log.Printf("CoinGecko: WARNING - Only fetched %d/%d requested coins due to rate limits",
			len(allCoins), totalLimit)
	}

	// Create final response
	return &APIResponse{
		Data: allCoins,
	}, nil
}

// fetchSinglePage fetches a single page of data from CoinGecko with retry capability
func (s *Service) fetchSinglePage(client *http.Client, page int, limit int) (*APIResponse, error) {
	// Retry configuration
	maxRetries := 3
	baseBackoff := 1000 * time.Millisecond

	var lastErr error

	// Try multiple times with exponential backoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Only log retry attempts after the first one
		if attempt > 0 {
			log.Printf("CoinGecko: Retry %d/%d for page %d after error: %v",
				attempt, maxRetries-1, page, lastErr)

			// Calculate backoff with jitter
			backoff := time.Duration(float64(baseBackoff) * math.Pow(2, float64(attempt-1)))
			jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
			sleepTime := backoff + jitter

			log.Printf("CoinGecko: Waiting %.2fs before retry", sleepTime.Seconds())
			time.Sleep(sleepTime)
		}

		// Get the appropriate base URL
		baseUrl := s.getApiBaseUrl()

		// Build URL with pagination parameters
		url := fmt.Sprintf("%s?vs_currency=usd&order=market_cap_desc&per_page=%d&page=%d",
			baseUrl,
			limit,
			page)

		// Add API key to URL if available
		if len(s.apiTokens.Tokens) > 0 {
			// Use the correct parameter name based on key type
			if s.isUsingDemoKey() {
				url = fmt.Sprintf("%s&x_cg_demo_api_key=%s", url, s.apiTokens.Tokens[0])
				if attempt == 0 {
					log.Printf("CoinGecko: Using Public API with Demo key for page %d request", page)
				}
			} else {
				url = fmt.Sprintf("%s&x_cg_pro_api_key=%s", url, s.apiTokens.Tokens[0])
				if attempt == 0 {
					log.Printf("CoinGecko: Using Pro API with Pro key for page %d request", page)
				}
			}
		} else if attempt == 0 {
			log.Printf("CoinGecko: No API key available, using public API for page %d", page)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("error creating request: %v", err)
			continue
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 Market-Proxy")

		// Start time for measuring request duration
		requestStart := time.Now()

		resp, err := client.Do(req)
		requestDuration := time.Since(requestStart)

		if err != nil {
			lastErr = fmt.Errorf("request failed after %.2fs: %v", requestDuration.Seconds(), err)
			continue
		}

		// Process response
		responseBody, err := processResponse(resp, page, requestDuration)
		if err != nil {
			// Check if we should retry this error or give up
			if isRetryableError(resp.StatusCode) {
				lastErr = err
				resp.Body.Close()
				continue
			}

			// For non-retryable errors, fail immediately
			resp.Body.Close()
			return nil, err
		}

		// Parse the JSON
		var coinsOriginal []CoinGeckoData
		if err := json.Unmarshal(responseBody, &coinsOriginal); err != nil {
			resp.Body.Close()
			// Parsing errors are not retryable
			return nil, fmt.Errorf("error parsing JSON for page %d: %v", page, err)
		}

		resp.Body.Close()

		// Convert to CoinMarketCap-compatible format
		coins := ConvertCoinGeckoData(coinsOriginal)

		// Log how many coins we received
		log.Printf("CoinGecko: Received %d coins from page %d in %.2fs", len(coins), page, requestDuration.Seconds())

		// Success - return the response
		return &APIResponse{
			Data: coins,
		}, nil
	}

	// If we got here, all retries failed
	return nil, fmt.Errorf("all %d attempts failed for page %d, last error: %v", maxRetries, page, lastErr)
}

// processResponse reads and processes the HTTP response
func processResponse(resp *http.Response, page int, requestDuration time.Duration) ([]byte, error) {
	// Check for rate limit or other errors
	if resp.StatusCode != http.StatusOK {
		// Read error body for more details
		body, _ := io.ReadAll(resp.Body)

		// Determine if this is a rate limit issue
		if resp.StatusCode == http.StatusTooManyRequests {
			// Check for retry-after header
			retryAfter := resp.Header.Get("Retry-After")
			return nil, fmt.Errorf("rate limit exceeded (status %d), retry after %s: %s",
				resp.StatusCode, retryAfter, string(body))
		}

		return nil, fmt.Errorf("API request failed with status %d after %.2fs: %s",
			resp.StatusCode, requestDuration.Seconds(), string(body))
	}

	// Try to read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Log response size
	log.Printf("CoinGecko: Page %d response size: %.2f KB", page, float64(len(responseBody))/1024)

	return responseBody, nil
}

// isRetryableError determines if a given HTTP status code should trigger a retry
func isRetryableError(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || // 429 Too Many Requests
		statusCode == http.StatusInternalServerError || // 500 Internal Server Error
		statusCode == http.StatusBadGateway || // 502 Bad Gateway
		statusCode == http.StatusServiceUnavailable || // 503 Service Unavailable
		statusCode == http.StatusGatewayTimeout // 504 Gateway Timeout
}

// fetchAndUpdate fetches data from CoinGecko and signals update
func (s *Service) fetchAndUpdate(ctx context.Context) error {
	data, err := s.fetchData()
	if err != nil {
		return err
	}

	s.cache.Lock()
	s.cache.data = data
	s.cache.Unlock()

	// Signal update through callback
	if s.onUpdate != nil {
		s.onUpdate()
	}

	return nil
}

func (s *Service) GetCacheData() *APIResponse {
	s.cache.RLock()
	defer s.cache.RUnlock()
	return s.cache.data
}
