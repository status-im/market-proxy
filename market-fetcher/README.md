# Market-Fetcher

Go application that caches token lists from CoinGecko and price updates from Binance, providing a REST API for accessing token and price data.

## Features

- Periodic updates of the complete token list from CoinGecko
- Real-time price updates from Binance WebSocket
- Configurable update intervals and token limits
- REST API endpoints for accessing token and price data
- Automatic reconnection handling for WebSocket connections
- Rate limit handling for CoinGecko API

## Local Development

### Running Locally

1. Create a configuration file `config.yaml`:
```yaml
coingecko_fetcher:
  update_interval: 10800  # seconds (3 hours)
  tokens_file: "coingecko_api_tokens.json"
  limit: 500  # number of tokens to fetch
```

2. Create `coingecko_api_tokens.json` (optional, for Pro API access):
```json
{
  "api_tokens": ["your-api-key-here"]
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

Trying to access the API:

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

### coingecko_fetcher

```yaml
coingecko_fetcher:
  update_interval_ms: 600000  # milliseconds (10 minutes)
  tokens_file: "coingecko_api_tokens.json"  # file containing API tokens
  limit: 5000  # number of tokens to fetch
  request_delay_ms: 0  # delay between requests, 0 = no delay
```

### coingecko_coinslist

```yaml
coingecko_coinslist:
  update_interval_ms: 1800000  # milliseconds (30 minutes)
  supported_platforms:  # list of blockchain platforms to include
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

#### coingecko_coinslist Parameters

- `update_interval_ms`: The interval in milliseconds at which to fetch token data
- `supported_platforms`: List of blockchain platforms to include in the token data. Tokens will be filtered to only include those available on the supported platforms.

### coingecko_api_tokens.json

API tokens configuration for CoinGecko:
```json
{
  "api_tokens": ["your-api-key-here"],
  "demo_api_tokens": ["demo-key"]
}
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