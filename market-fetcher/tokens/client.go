package tokens

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	// Default URL for CoinGecko API
	DefaultCoinGeckoBaseURL = "https://api.coingecko.com"
	// Endpoint for coin list with platforms
	CoinsListEndpoint = "/api/v3/coins/list?include_platform=true"
	// Timeout for HTTP requests
	requestTimeout = 30 * time.Second
)

// Client handles HTTP communication with the CoinGecko API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultCoinGeckoBaseURL
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// FetchTokens retrieves the list of tokens from CoinGecko API
func (c *Client) FetchTokens() ([]Token, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, CoinsListEndpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var tokens []Token
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return tokens, nil
}
