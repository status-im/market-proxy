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

### config.yaml

Main configuration file for the service:
```yaml
coingecko_fetcher:
  update_interval: 10800  # seconds between CoinGecko updates
  tokens_file: "coingecko_api_tokens.json"  # path to API tokens file
  limit: 500  # maximum number of tokens to fetch
```

### coingecko_api_tokens.json

API tokens configuration for CoinGecko:
```json
{
  "api_tokens": ["your-api-key-here"]
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
3. Health check available via `/health`

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