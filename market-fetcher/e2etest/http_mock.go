package e2etest

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Add this syncWSConn struct to encapsulate a WebSocket connection with mutex protection
type syncWSConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// MockServer represents a mock server for testing HTTP and WebSocket requests
type MockServer struct {
	server         *httptest.Server
	ExchangeData   *ExchangeMockData
	CoinGeckoData  *CoinGeckoMockData
	CoinGeckoMock  *CoinGeckoMock
	BinanceMock    *BinanceMock
	WebSocketPath  string
	upgrader       websocket.Upgrader
	mu             sync.RWMutex // Mutex to protect websocketConns
	websocketConns []*syncWSConn
}

// CoinGeckoMock contains mock data for CoinGecko API
type CoinGeckoMock struct {
	LeaderboardData string // JSON data for the leaderboard
	TokensListData  string // JSON data for the token list
}

// BinanceMock contains mock data for Binance API
type BinanceMock struct {
	PriceData string // JSON data for prices
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
// addr - optional parameter to specify the server address (ignored when using httptest.Server)
func NewMockServer(addr ...string) *MockServer {
	ms := &MockServer{
		ExchangeData:  NewExchangeMockData(),
		CoinGeckoData: NewCoinGeckoMockData(),
		CoinGeckoMock: &CoinGeckoMock{
			LeaderboardData: defaultLeaderboardData(),
			TokensListData:  defaultTokensListData(),
		},
		BinanceMock: &BinanceMock{
			PriceData: defaultBinancePriceData(),
		},
		WebSocketPath: "/ws/!ticker@arr",
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		websocketConns: make([]*syncWSConn, 0),
	}

	// Create HTTP server with request handler
	mux := http.NewServeMux()
	mux.HandleFunc("/", ms.handleRequest)

	// Create server
	// Note: httptest.Server automatically selects a free port
	server := httptest.NewServer(mux)
	ms.server = server

	// Start goroutine to send test data to WebSocket clients
	go ms.broadcastPriceUpdates()

	return ms
}

// Close closes the mock server and all WebSocket connections
func (ms *MockServer) Close() {
	// Close all WebSocket connections
	ms.mu.Lock()
	for _, syncConn := range ms.websocketConns {
		syncConn.mu.Lock()
		syncConn.conn.Close()
		syncConn.mu.Unlock()
	}
	// Clear the connections list
	ms.websocketConns = nil
	ms.mu.Unlock()

	if ms.server != nil {
		ms.server.Close()
	}
}

// handleRequest processes incoming requests and returns mock data
func (ms *MockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	query := r.URL.Query()

	log.Printf("MockServer: Received request for path: %s", path)

	// WebSocket connection handling - any path containing /ws/
	if strings.Contains(path, "/ws/") {
		log.Printf("MockServer: Detected WebSocket path in request: %s", path)
		ms.handleWebSocket(w, r)
		return
	}

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

	// Binance API mock responses
	if strings.Contains(path, "/api/v3/ticker/price") {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, ms.BinanceMock.PriceData)
		return
	}

	// Return 404 for unknown paths
	log.Printf("MockServer: Path not found: %s", path)
	http.NotFound(w, r)
}

// handleWebSocket handles WebSocket connection requests
func (ms *MockServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("MockServer: Received WebSocket connection request at path: %s", r.URL.Path)

	conn, err := ms.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS Error: Could not open websocket connection: %v", err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}

	log.Printf("MockServer: Successfully established WebSocket connection")

	// Create a synchronized wrapper for the connection
	syncConn := &syncWSConn{conn: conn}

	// Add new connection to list
	ms.mu.Lock()
	ms.websocketConns = append(ms.websocketConns, syncConn)
	ms.mu.Unlock()

	// Send initial data right after connection (with mutex protection)
	syncConn.mu.Lock()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(ms.BinanceMock.PriceData)); err != nil {
		log.Printf("WS Error: Failed to send initial data: %v", err)
		syncConn.mu.Unlock()
	} else {
		log.Printf("WS: Initial data sent successfully. First connection: %v", conn.LocalAddr())
		syncConn.mu.Unlock()
	}
}

// broadcastPriceUpdates sends periodic price updates via WebSocket
func (ms *MockServer) broadcastPriceUpdates() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Create a copy of the connections slice to avoid issues with parallel modifications
		ms.copyAndBroadcast()
	}
}

// copyAndBroadcast creates a copy of connections and sends them data
func (ms *MockServer) copyAndBroadcast() {
	// Take a read lock to create a copy of connections
	ms.mu.RLock()
	connections := make([]*syncWSConn, len(ms.websocketConns))
	copy(connections, ms.websocketConns)
	ms.mu.RUnlock()

	// Send data to all connected clients (outside the lock)
	for i, syncConn := range connections {
		if syncConn == nil {
			continue
		}

		// Lock the individual connection before writing
		syncConn.mu.Lock()
		err := syncConn.conn.WriteMessage(websocket.TextMessage, []byte(ms.BinanceMock.PriceData))
		syncConn.mu.Unlock()

		if err != nil {
			log.Printf("WS Error: Failed to send data: %v", err)

			// Remove the failed connection from main list
			ms.removeConnection(syncConn)
		} else if i == 0 {
			// Log only for the first connection to avoid cluttering the output
			log.Printf("WS: Broadcast data sent successfully to %d connections", len(connections))
		}
	}
}

// removeConnection removes a closed connection from the websocketConns slice
func (ms *MockServer) removeConnection(syncConn *syncWSConn) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for j, c := range ms.websocketConns {
		if c == syncConn {
			// Remove the connection from the slice
			ms.websocketConns = append(ms.websocketConns[:j], ms.websocketConns[j+1:]...)

			// Close the connection (with mutex protection)
			syncConn.mu.Lock()
			syncConn.conn.Close()
			syncConn.mu.Unlock()
			break
		}
	}
}

// GetURL returns the base URL of the mock server
func (ms *MockServer) GetURL() string {
	return ms.server.URL
}

// GetWSURL returns the WebSocket URL of the mock server
func (ms *MockServer) GetWSURL() string {
	// Replace http with ws in URL
	wsURL := "ws" + strings.TrimPrefix(ms.server.URL, "http") + ms.WebSocketPath
	log.Printf("MockServer: WebSocket URL is %s", wsURL)
	return wsURL
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

// defaultBinancePriceData returns test data for Binance prices
func defaultBinancePriceData() string {
	return `[
		{
			"e": "24hrTicker", 
			"E": 1587558973622,
			"s": "BTCUSDT",
			"p": "1000.00",
			"P": "2.00",
			"w": "49000.00",
			"c": "50000.00",
			"Q": "1.00000000",
			"b": "49900.00",
			"B": "1.00000000",
			"a": "50100.00",
			"A": "1.00000000",
			"o": "49000.00",
			"h": "51000.00",
			"l": "49000.00",
			"v": "100000.00000000",
			"q": "4950000000.00000000",
			"O": 1587472573622,
			"C": 1587558973622,
			"F": 100,
			"L": 200,
			"n": 100
		},
		{
			"e": "24hrTicker", 
			"E": 1587558973622,
			"s": "ETHUSDT",
			"p": "100.00",
			"P": "3.33",
			"w": "2900.00",
			"c": "3000.00",
			"Q": "1.00000000",
			"b": "2950.00",
			"B": "1.00000000",
			"a": "3050.00",
			"A": "1.00000000",
			"o": "2900.00",
			"h": "3100.00",
			"l": "2900.00",
			"v": "50000.00000000",
			"q": "150000000.00000000",
			"O": 1587472573622,
			"C": 1587558973622,
			"F": 300,
			"L": 400,
			"n": 100
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
	// Add proxy routing for test API endpoints
	// We could implement request redirection here,
	// but in the current architecture it's not required since
	// endpoints are already handled in handleRequest

	log.Printf("MockServer: Added proxy rules for real server at %s", serverBaseURL)
	log.Printf("MockServer: Mock server running at %s", ms.server.URL)
	log.Printf("MockServer: WebSocket URL at %s", ms.GetWSURL())
}
