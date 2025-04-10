# Market Proxy

[![Tests](https://github.com/status-im/market-proxy/actions/workflows/test.yml/badge.svg)](https://github.com/status-im/market-proxy/actions/workflows/test.yml)

A Go-based market data fetcher agent with caching and an Nginx proxy for efficient data delivery.

## Overview

This project consists of two main components:

1. **Market Fetcher**: A Go service that fetches and caches market data from CoinGecko and price updates from Binance.
2. **Nginx Proxy**: A reverse proxy that provides caching, ETag optimization, and compression for efficient delivery of market data.

## Local Development

### Prerequisites

- Docker and Docker Compose


### Configuration

1. Create a `config.yaml` file in the `market-fetcher` directory:
```yaml
coingecko_fetcher:
  update_interval: 10800  # seconds (3 hours)
  tokens_file: "coingecko_api_tokens.json"
  limit: 500  # number of tokens to fetch
```

2. (Optional) Create `coingecko_api_tokens.json` in the `secrets` directory for Pro API access:
```json
{
  "api_tokens": ["your-api-key-here"]
}
```

If you don't provide this file, the service will use the public API without authentication.

### Running Locally

Run the following command to start all services:

```bash
./start-local.sh
```

This will:
1. Create necessary configuration files if they don't exist
2. Build and start the following services:
   - **market-fetcher**: Fetches market data (port 8081)
   - **market-proxy**: Nginx proxy with caching (port 8080)
   - **market-frontend**: Test frontend application (port 3000)
3. Set up a Docker network for communication between services

### Accessing the Services

- API Proxy: http://localhost:8080
- Frontend: http://localhost:3000

![img.png](test-api.png)

## Subprojects

### [Market Fetcher](./market-fetcher/README.md)

Go application that caches token lists from CoinGecko and price updates from Binance, providing a REST API for accessing token and price data.

### [Nginx Proxy](./nginx-proxy/README.md)

A proxy that performs caching, ETag optimization, and compression for displaying market data from market-fetcher.
