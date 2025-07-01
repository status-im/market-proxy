# Market-Fetcher

Go application that provides cached cryptocurrency data via REST API. The service fetches token lists, market data, and price updates from CoinGecko, and offers CoinGecko-compatible endpoints with intelligent caching.

## Features

- Periodic updates of the complete token list from CoinGecko
- Periodic updates of top market data from CoinGecko (leaderboard)
- Periodic updates of top token prices from CoinGecko (leaderboard)
- CoinGecko-compatible `/api/v3/coins/markets` endpoint with pagination
- CoinGecko-compatible `/api/v3/simple/price` endpoint with caching
- Configurable update intervals, token limits, and cache TTLs
- REST API endpoints for accessing token, market, and price data
- Rate limit handling for CoinGecko API
- Monitoring and health checks 

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
  top_markets_update_interval: 5m  # Market data refresh interval
  top_markets_limit: 100           # Number of top tokens to fetch
  currency: usd                    # Currency for market data
  top_prices_update_interval: 2m   # Price refresh interval
  top_prices_limit: 50             # Tokens for price tracking

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

# Markets service with caching
coingecko_markets:
  chunk_size: 250              # Tokens per API request (max 250)
  request_delay: 200ms         # Delay between requests
  ttl: 5m                      # Market data cache TTL

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
  top_markets_update_interval: 5m  # How often to refresh top markets data
  top_markets_limit: 100           # Number of top tokens to fetch
  currency: usd                    # Currency for market data (usd, eur, etc.)
  top_prices_update_interval: 2m   # How often to update price data
  top_prices_limit: 50             # Number of tokens to track for prices
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

#### CoinGecko Markets Service

```yaml
coingecko_markets:
  chunk_size: 250              # Tokens per API request (max 250)
  request_delay: 200ms         # Delay between requests
  ttl: 5m                      # Cache TTL for market data
```
## Request Flow

### Top Markets Updates
1. The top markets updater runs at configured intervals (`top_markets_update_interval`)
2. Fetches top market data from CoinGecko API
3. Updates the markets cache with top tokens
4. Triggers top prices update with token IDs from markets data

### Top Prices Updates
1. The top prices updater runs at configured intervals (`top_prices_update_interval`)
2. Fetches price data for top tokens from CoinGecko API
3. Updates the prices cache with latest data
4. Token list comes from top markets data

### Markets Service
1. Provides CoinGecko-compatible `/api/v3/coins/markets` endpoint
2. Supports pagination, filtering, and sorting parameters
3. Uses intelligent caching with configurable TTL
4. Handles chunked requests for large datasets

### Prices Service
1. Provides CoinGecko-compatible `/api/v3/simple/price` endpoint  
2. Supports multiple currencies and optional fields
3. Uses intelligent caching with configurable TTL
4. Handles chunked requests for large token lists

### REST API Access
1. Top markets data available via `/api/v1/leaderboard/markets`
2. Top prices data available via `/api/v1/leaderboard/prices`
3. Top simple prices available via `/api/v1/leaderboard/simpleprices`
4. CoinGecko-compatible markets via `/api/v1/coins/markets`
5. CoinGecko-compatible simple prices via `/api/v1/simple/price`
6. Token platform data available via `/api/v1/coins/list`
7. Health check available via `/health`
8. Prometheus metrics available via `/metrics`

## API Endpoints

### GET /api/v1/leaderboard/prices
Returns latest price data for top tokens:
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

### GET /api/v1/leaderboard/simpleprices
Returns simple price data for top tokens in specified currency:
```bash
# Query parameters: ?currency=usd
```
```json
{
  "bitcoin": {
    "usd": 50000.00
  },
  "ethereum": {
    "usd": 3000.00
  }
}
```

### GET /api/v1/leaderboard/markets
Returns top market data from CoinGecko:
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

### GET /api/v1/coins/markets
CoinGecko-compatible markets endpoint with pagination and filtering:
```bash
# Query parameters: ?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false
```
```json
[
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
```

### GET /api/v1/simple/price
CoinGecko-compatible simple price endpoint:
```bash
# Query parameters: ?ids=bitcoin,ethereum&vs_currencies=usd,eur&include_market_cap=true
```
```json
{
  "bitcoin": {
    "usd": 50000.00,
    "eur": 42000.00,
    "usd_market_cap": 1000000000000
  },
  "ethereum": {
    "usd": 3000.00,
    "eur": 2500.00,
    "usd_market_cap": 360000000000
  }
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
    "coingecko": "up",
    "tokens": "up",
    "coingecko_prices": "up",
    "coingecko_markets": "up"
}
}
```

## Environment Variables

- `PORT` - HTTP server port (default: 8080)