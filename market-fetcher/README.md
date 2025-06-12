# Market-Fetcher

Go application that provides cached cryptocurrency data with real-time price updates via REST API. The service fetches token lists and market data from CoinGecko, price updates from Binance WebSocket, and offers CoinGecko-compatible endpoints with intelligent caching.

## Features

- Periodic updates of the complete token list from CoinGecko
- Real-time price updates from Binance WebSocket
- Configurable update intervals and token limits
- REST API endpoints for accessing token and price data
- Rate limit handling for CoinGecko API
- Monitoring 

## Local Development

### Running Locally

1. Create a configuration file `config.yaml`:
```yaml
# Cache configuration
cache:
  go_cache:
    default_expiration: 10m    # Default cache TTL
    cleanup_interval: 5m       # Cache cleanup frequency

# Token list fetcher
coingecko_coinslist:
  update_interval: 30m
  supported_platforms:
    - ethereum
    - optimistic-ethereum
    - arbitrum-one
    - base
    - polygon-pos
    - binance-smart-chain

# Market data fetcher  
coingecko_leaderboard:
  update_interval: 5m
  limit: 100                   # Number of top tokens
  request_delay: 1s            # Delay between requests
  prices_update_interval: 2m   # Price refresh interval
  top_tokens_limit: 50         # Tokens for price tracking

# Price service with caching
coingecko_prices:
  chunk_size: 250              # Tokens per API request
  request_delay: 200ms         # Delay between chunks
  ttl: 2m                      # Price cache TTL
  currencies:                  # Default currencies to cache
    - usd
    - eur
    - btc
    - eth

# API tokens file
tokens_file: "coingecko_api_tokens.json"
```

2. Create `coingecko_api_tokens.json` (optional, for Pro API access):
```json
{
  "api_tokens": ["your-coingecko-api-key"],
  "demo_api_tokens": ["demo-key"]
}
```

3. Run the application:
```bash
go run main.go
```

### Using Docker

Build and run the container:
```bash
./build_docker_locally_run.sh
```

The service will be available at `http://localhost:8080` by default.

To access the API:

```
curl http://localhost:8080/api/v1/leaderboard/markets
```
## Deployment

### Docker

1. Build the container:
```bash
docker build -t market-fetcher .
```

2. Run the container:
```bash
docker run -p 8080:8080 market-fetcher
```

## Configuration

The service is configured via a `config.yaml` file. Below are the key configuration sections:

#### Cache Configuration

```yaml
cache:
  go_cache:
    default_expiration: 10m    # Default TTL for cached items
    cleanup_interval: 5m       # How often to clean up expired items
```
#### CoinGecko Tokens Service

```yaml
coingecko_coinslist:
  update_interval: 30m         # How often to refresh token list
  supported_platforms:         # Blockchain platforms to include
    - ethereum
    - optimistic-ethereum  # Optimism
    - arbitrum-one         # Arbitrum
    - base
    - status
    - linea
    - blast
    - zksync
    - mantle
    - abstract
    - unichain
    - binance-smart-chain  # BSC
    - polygon-pos          # Polygon
```

#### CoinGecko Leaderboard Service

```yaml
coingecko_leaderboard:
  update_interval: 5m          # Market data refresh interval
  limit: 100                   # Number of top tokens to fetch
  request_delay: 1s            # Delay between API requests
  prices_update_interval: 2m   # How often to update price data
  top_tokens_limit: 50         # Tokens to track for prices
```

#### CoinGecko Prices Service

```yaml
coingecko_prices:
  chunk_size: 250              # Tokens per API request (max 250)
  request_delay: 200ms         # Delay between chunk requests
  ttl: 2m                      # Cache TTL for price data
  currencies:                  # Default currencies to cache
    - usd
    - eur
    - btc
    - eth
```
## Request Flow

### Token List Updates
1. The scheduler task runs at configured intervals
2. Fetches token list from CoinGecko API
3. Updates the token cache
4. Triggers Binance watchlist update with new tokens

### Price Updates
1. Binance WebSocket connection maintained for real-time updates
2. Price updates received for watched symbols
3. Price cache updated with latest data
4. Price cache reset when token list is updated

### REST API Access
1. Token data available via `/api/v1/leaderboard/markets`
2. Price data available via `/api/v1/leaderboard/prices`
3. Token platform data available via `/api/v1/coins/list`
4. Health check available via `/health`

## API Endpoints

### GET /api/v1/leaderboard/prices
Returns latest price data for all watched tokens:
```json
{
  "BTC": {
    "price": 50000.00,
    "percent_change_24h": 1.5
  },
  "ETH": {
    "price": 3000.00,
    "percent_change_24h": -0.5
  }
}
```

### GET /api/v1/leaderboard/markets
Returns token market data from CoinGecko:
```json
{
  "data": [
  {
    "id": "bitcoin",
    "symbol": "btc", 
    "name": "Bitcoin",
      "image": "https://...",
      "current_price": 50000.00,
      "market_cap": 1000000000000,
      "total_volume": 50000000000,
      "price_change_percentage_24h": 1.5
    }
  ]
}
```

### GET /api/v1/coins/list
Returns a list of all tokens with their supported blockchain platforms:
```json
[
  {
    "id": "ethereum",
    "symbol": "eth",
    "name": "Ethereum", 
    "platforms": {
      "ethereum": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
  }
  },
  {
    "id": "status",
    "symbol": "snt",
    "name": "Status",
    "platforms": {
      "ethereum": "0x744d70fdbe2ba4cf95131626614a1763df805b9e",
      "status": "0x744d70fdbe2ba4cf95131626614a1763df805b9e"
  }
}
]
```

### GET /health
Returns service health status:
```json
{
  "status": "ok",
  "services": {
    "binance": "up",
    "coingecko": "up"
}
}
```

## Environment Variables

- `PORT` - HTTP server port (default: 8080)