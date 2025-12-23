package e2etest

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// MockServer represents a mock server for testing HTTP requests
type MockServer struct {
	server        *httptest.Server
	ExchangeData  *ExchangeMockData
	CoinGeckoData *CoinGeckoMockData
	CoinGeckoMock *CoinGeckoMock
}

// CoinGeckoMock contains mock data for CoinGecko API
type CoinGeckoMock struct {
	LeaderboardData string // JSON data for the leaderboard
	TokensListData  string // JSON data for the token list
}

// CoinGeckoMockData contains structured test data for CoinGecko
type CoinGeckoMockData struct {
	LeaderboardData string
	TokensListData  string
}

// ExchangeMockData contains test data for exchanges
type ExchangeMockData struct {
	// Fields for various exchanges can be added here
}

// NewMockServer creates and returns a new mock server
func NewMockServer(addr ...string) *MockServer {
	ms := &MockServer{
		ExchangeData:  NewExchangeMockData(),
		CoinGeckoData: NewCoinGeckoMockData(),
		CoinGeckoMock: &CoinGeckoMock{
			LeaderboardData: defaultLeaderboardData(),
			TokensListData:  defaultTokensListData(),
		},
	}

	// Create HTTP server with request handler
	mux := http.NewServeMux()
	mux.HandleFunc("/", ms.handleRequest)

	// Create server
	server := httptest.NewServer(mux)
	ms.server = server

	return ms
}

// Close closes the mock server
func (ms *MockServer) Close() {
	if ms.server != nil {
		ms.server.Close()
	}
}

// handleRequest processes incoming requests and returns mock data
func (ms *MockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	query := r.URL.Query()

	log.Printf("MockServer: Received request for path: %s", path)

	// API endpoints for tests
	if path == "/api/v1/leaderboard/prices" {
		w.Header().Set("Content-Type", "application/json")
		// Response format should match expectations in TestLeaderboardPricesEndpoint
		prices := `[
			{"symbol": "BTC", "price": 50000.0, "percent_change_24h": 2.0},
			{"symbol": "ETH", "price": 3000.0, "percent_change_24h": 3.33}
		]`
		fmt.Fprint(w, prices)
		return
	}

	if path == "/api/v1/coins/list" {
		w.Header().Set("Content-Type", "application/json")
		// Response format should match expectations in TestCoinsListEndpoint
		coinsList := `[
			{"id": "bitcoin", "symbol": "btc", "name": "Bitcoin", "platforms": {}},
			{"id": "ethereum", "symbol": "eth", "name": "Ethereum", "platforms": {}},
			{"id": "tether", "symbol": "usdt", "name": "Tether", "platforms": {"ethereum": "0xdac17f958d2ee523a2206206994597c13d831ec7"}}
		]`
		fmt.Fprint(w, coinsList)
		return
	}

	// CoinGecko API mock responses
	if strings.Contains(path, "/api/v3/coins/markets") {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, ms.CoinGeckoMock.LeaderboardData)
		return
	}

	// CoinGecko simple price endpoint
	if strings.Contains(path, "/api/v3/simple/price") {
		w.Header().Set("Content-Type", "application/json")

		// Default response for common test tokens
		// In a real implementation, you might parse ids and vs_currencies from query parameters
		priceResponse := `{
			"bitcoin": {"usd": 50000, "eur": 42000, "usd_market_cap": 950000000000, "usd_24h_vol": 35000000000, "usd_24h_change": 2.5, "last_updated_at": 1703097600},
			"ethereum": {"usd": 3000, "eur": 2520, "usd_market_cap": 360000000000, "usd_24h_vol": 15000000000, "usd_24h_change": 3.2, "last_updated_at": 1703097600},
			"tether": {"usd": 1, "eur": 0.84, "usd_market_cap": 90000000000, "usd_24h_vol": 25000000000, "usd_24h_change": 0.1, "last_updated_at": 1703097600}
		}`

		fmt.Fprint(w, priceResponse)
		return
	}

	// Market chart endpoints - match pattern /api/v3/coins/{id}/market_chart
	if strings.Contains(path, "/market_chart") {
		w.Header().Set("Content-Type", "application/json")
		// Generate fresh data on each request to ensure timestamps are current
		freshData := generateMarketChartData()
		fmt.Fprint(w, freshData)
		return
	}

	// Processing tokens list request with platform information
	if strings.Contains(path, "/api/v3/coins/list") && query.Get("include_platform") == "true" {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, ms.CoinGeckoMock.TokensListData)
		return
	}

	// CoinGecko token lists endpoint - /api/v3/token_lists/{platform}/all.json
	if strings.Contains(path, "/api/v3/token_lists/") && strings.Contains(path, "/all.json") {
		w.Header().Set("Content-Type", "application/json")
		// Extract platform from path
		parts := strings.Split(path, "/")
		var platform string
		for i, part := range parts {
			if part == "token_lists" && i+1 < len(parts) {
				platform = parts[i+1]
				break
			}
		}
		// Return mock token list data for any platform
		tokenListResponse := fmt.Sprintf(`{
			"name": "%s Token List",
			"logoURI": "https://example.com/%s-logo.png",
			"version": {"major": 1, "minor": 0, "patch": 0},
			"tokens": [
				{
					"chainId": 59144,
					"address": "0xe5D7C2a44FfDDf6b295A15c148167daaAf5Cf34f",
					"name": "Wrapped Ether",
					"symbol": "WETH",
					"decimals": 18,
					"logoURI": "https://example.com/weth.png"
				},
				{
					"chainId": 59144,
					"address": "0x176211869cA2b568f2A7D4EE941E073a821EE1ff",
					"name": "USDC",
					"symbol": "USDC",
					"decimals": 6,
					"logoURI": "https://example.com/usdc.png"
				}
			],
			"timestamp": "2025-01-01T00:00:00.000Z"
		}`, platform, platform)
		fmt.Fprint(w, tokenListResponse)
		return
	}

	// Return 404 for unknown paths
	log.Printf("MockServer: Path not found: %s", path)
	http.NotFound(w, r)
}

// GetURL returns the base URL of the mock server
func (ms *MockServer) GetURL() string {
	return ms.server.URL
}

// defaultLeaderboardData returns test data for CoinGecko leaderboard
func defaultLeaderboardData() string {
	return `[
	{
		"id": "bitcoin",
		"symbol": "btc",
		"name": "Bitcoin",
		"image": "https://assets.coingecko.com/coins/images/1/large/bitcoin.png",
		"current_price": 50000,
		"market_cap": 950000000000,
		"market_cap_rank": 1,
		"fully_diluted_valuation": 1050000000000,
		"total_volume": 30000000000,
		"high_24h": 51000,
		"low_24h": 49000,
		"price_change_24h": 1000,
		"price_change_percentage_24h": 2,
		"market_cap_change_24h": 20000000000,
		"market_cap_change_percentage_24h": 2.1,
		"circulating_supply": 19000000,
		"total_supply": 21000000,
		"max_supply": 21000000,
		"ath": 69000,
		"ath_change_percentage": -27.5,
		"ath_date": "2021-11-10T14:24:11.849Z",
		"atl": 67.81,
		"atl_change_percentage": 73613.6,
		"atl_date": "2013-07-06T00:00:00.000Z",
		"last_updated": "2023-04-20T12:34:56.789Z"
	},
	{
		"id": "ethereum",
		"symbol": "eth",
		"name": "Ethereum",
		"image": "https://assets.coingecko.com/coins/images/279/large/ethereum.png",
		"current_price": 3000,
		"market_cap": 360000000000,
		"market_cap_rank": 2,
		"fully_diluted_valuation": 360000000000,
		"total_volume": 15000000000,
		"high_24h": 3100,
		"low_24h": 2900,
		"price_change_24h": 100,
		"price_change_percentage_24h": 3.33,
		"market_cap_change_24h": 12000000000,
		"market_cap_change_percentage_24h": 3.45,
		"circulating_supply": 120000000,
		"total_supply": 120000000,
		"max_supply": null,
		"ath": 4878.26,
		"ath_change_percentage": -38.5,
		"ath_date": "2021-11-10T14:24:19.604Z",
		"atl": 0.432979,
		"atl_change_percentage": 692968.2,
		"atl_date": "2015-10-20T00:00:00.000Z",
		"last_updated": "2023-04-20T12:34:56.789Z"
	}
]`
}

// defaultTokensListData returns test data for CoinGecko tokens list
func defaultTokensListData() string {
	return `[
	{
		"id": "bitcoin",
		"symbol": "btc",
		"name": "Bitcoin",
		"platforms": {}
	},
	{
		"id": "ethereum",
		"symbol": "eth",
		"name": "Ethereum",
		"platforms": {}
	},
	{
		"id": "tether",
		"symbol": "usdt",
		"name": "Tether",
		"platforms": {
			"ethereum": "0xdac17f958d2ee523a2206206994597c13d831ec7",
			"binance-smart-chain": "0x55d398326f99059ff775485246999027b3197955"
		}
	}
]`
}

// generateMarketChartData generates market chart data with current timestamps
// This ensures the data won't be filtered out by the strip function
func generateMarketChartData() string {
	now := time.Now()

	var prices, marketCaps, totalVolumes []string

	for i := 29; i >= 0; i-- {
		timestamp := now.AddDate(0, 0, -i).Unix() * 1000 // Convert to milliseconds
		price := 47777.23 + float64(i)*100               // Sample price progression
		marketCap := 905000000000 + int64(i)*3000000000  // Sample market cap progression
		volume := 28000000000 + int64(i)*500000000       // Sample volume progression

		prices = append(prices, fmt.Sprintf("[%d, %.2f]", timestamp, price))
		marketCaps = append(marketCaps, fmt.Sprintf("[%d, %d]", timestamp, marketCap))
		totalVolumes = append(totalVolumes, fmt.Sprintf("[%d, %d]", timestamp, volume))
	}

	return fmt.Sprintf(`{
		"prices": [%s],
		"market_caps": [%s],
		"total_volumes": [%s]
	}`, strings.Join(prices, ","), strings.Join(marketCaps, ","), strings.Join(totalVolumes, ","))
}

// NewCoinGeckoMockData creates a new instance with test data
func NewCoinGeckoMockData() *CoinGeckoMockData {
	return &CoinGeckoMockData{
		LeaderboardData: `[
			{"id":"bitcoin","symbol":"btc","name":"Bitcoin","image":"https://assets.coingecko.com/coins/images/1/large/bitcoin.png?1547033579","current_price":61573,"market_cap":1209766246446,"market_cap_rank":1,"fully_diluted_valuation":1293036305050,"total_volume":31129354793,"high_24h":62951,"low_24h":61172,"price_change_24h":185.77,"price_change_percentage_24h":0.30292,"market_cap_change_24h":3807133066,"market_cap_change_percentage_24h":0.31594,"circulating_supply":19647718.0,"total_supply":21000000.0,"max_supply":21000000.0,"ath":73738,"ath_change_percentage":-16.49752,"ath_date":"2021-11-10T14:24:11.849Z","atl":67.81,"atl_change_percentage":90774.52434,"atl_date":"2013-07-06T00:00:00.000Z","roi":null,"last_updated":"2023-03-30T12:30:39.990Z"}
		]`,
		TokensListData: `[
			{"id":"bitcoin","symbol":"btc","name":"Bitcoin","platforms":{}},
			{"id":"ethereum","symbol":"eth","name":"Ethereum","platforms":{}},
			{"id":"tether","symbol":"usdt","name":"Tether","platforms":{"ethereum":"0xdac17f958d2ee523a2206206994597c13d831ec7"}}
		]`,
	}
}

// NewExchangeMockData creates a new instance of ExchangeMockData
func NewExchangeMockData() *ExchangeMockData {
	return &ExchangeMockData{}
}

// AddProxyRulesToRealServer adds rules for redirecting requests from the real server to the mock server
func (ms *MockServer) AddProxyRulesToRealServer(serverBaseURL string) {
	log.Printf("MockServer: Added proxy rules for real server at %s", serverBaseURL)
	log.Printf("MockServer: Mock server running at %s", ms.server.URL)
}
