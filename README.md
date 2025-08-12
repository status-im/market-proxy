# Market Proxy

[![Tests](https://github.com/status-im/market-proxy/actions/workflows/test.yml/badge.svg)](https://github.com/status-im/market-proxy/actions/workflows/test.yml)

A Go-based market data fetcher agent with caching and an Nginx proxy for efficient data delivery.

## Overview

This project consists of two main components:

1. **Market Fetcher**: A Go service that fetches and caches market data, token lists, and price updates from CoinGecko, providing CoinGecko-compatible REST APIs with intelligent caching and blockchain platform-specific token filtering.
2. **Nginx Proxy**: A reverse proxy that provides caching, ETag optimization, and compression for efficient delivery of market data.

## Local Development

### Prerequisites

- Docker and Docker Compose


### Configuration

1. Create a `config.yaml` file in the `market-fetcher` directory:
```yaml
# Cache configuration
cache:
  go_cache:
    default_expiration: 5m     # Default cache TTL
    cleanup_interval: 10m      # Cache cleanup frequency

# Token list fetcher
coingecko_coinslist:
  update_interval: 30m
  supported_platforms:
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

# Leaderboard service (top markets and prices)
coingecko_leaderboard:
  top_markets_update_interval: 30m # Market data refresh interval
  top_markets_limit: 5000          # Number of top tokens to fetch
  currency: usd                    # Currency for market data
  top_prices_update_interval: 30s  # Price refresh interval
  top_prices_limit: 500            # Tokens for price tracking

# Tier-based prices service with intelligent caching
coingecko_prices:
  chunk_size: 500              # Tokens per API request
  request_delay: 300ms         # Delay between chunks
  ttl: 10m                     # Default price cache TTL
  currencies:                  # Default currencies to cache
    - usd
    - eur
    - btc
    - eth
  tiers:                       # Tier-based update configuration
    - name: "top-1000"         # High-frequency tier for top tokens
      token_from: 1            # Top 1000 tokens
      token_to: 1000
      update_interval: 30s     # Update every 30 seconds
    - name: "top-1001-5000"    # Medium-frequency tier for mid-range tokens
      token_from: 1001         # from 1001-5000
      token_to: 5000
      update_interval: 5m      # Update every 5 minutes
      fetch_coinslist_ids: true # fetch missing tokens from supported platforms

# Tier-based markets service with intelligent caching
coingecko_markets:
  request_delay: 300ms         # Delay between requests
  ttl: 35m                     # Default market data cache TTL
  market_params_normalize:     # Normalize parameters for consistent caching
    vs_currency: "usd"         # Override currency to USD
    order: "market_cap_desc"   # Override order to market cap descending
    per_page: 250              # Override per_page to maximum
    sparkline: false           # Override sparkline to false
    price_change_percentage: "1h,24h"  # Override price changes to 1h,24h
    category: ""               # Override category to empty (no filtering)
  tiers:                       # Tier-based update configuration
    - name: "top-500"          # High-frequency tier for top tokens
      page_from: 1             # Top 500 tokens (pages 1-2 with per_page: 250)
      page_to: 2
      update_interval: 30s     # Update every 30 seconds
    - name: "top-501-5000"     # Medium-frequency tier for mid-range tokens
      page_from: 3             # from 501-5000 (pages 3-20)
      page_to: 20
      update_interval: 30m     # Update every 30 minutes
      fetch_coinslist_ids: true # fetch missing tokens from supported platforms

# Market chart service with intelligent caching
coingecko_market_chart:
  hourly_ttl: 30m             # TTL for hourly data (requests with days <= daily_data_threshold)
  daily_ttl: 12h              # TTL for daily data (requests with days > daily_data_threshold)  
  daily_data_threshold: 90    # threshold in days: <= 90 days = hourly data, > 90 days = daily data
  try_free_api_first: true    # try free API (no key) first when no interval is specified

# API tokens file
tokens_file: "coingecko_api_tokens.json"
```

2. (Optional) Create `coingecko_api_tokens.json` in the `secrets` directory for Pro API access:
```json
{
   "api_tokens": ["your-api-key-here"], 
   "demo_api_tokens": ["demo-key"]
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

## Key Features

- Tier-based intelligent caching system with different update intervals for various token ranges
- High-frequency updates for top 500-1000 tokens (30 seconds) and lower frequency for less popular tokens
- CoinGecko-compatible `/api/v3/coins/markets` endpoint with pagination and tier-based caching
- CoinGecko-compatible `/api/v3/simple/price` endpoint with tier-based price caching
- Market chart service with intelligent caching and request enrichment
- Blockchain platform-specific token filtering for supported networks
- Event-driven subscription system for real-time data updates
- Advanced metrics and monitoring with Prometheus integration
- REST API for accessing token lists, market data, and prices with cache status headers
- Rate limit handling and retry mechanisms for CoinGecko API
- Health checks and comprehensive monitoring

## API Endpoints

The proxy provides the following endpoints:

- `/v1/simple/price` - CoinGecko-compatible simple price endpoint
- `/v1/coins/markets` - CoinGecko-compatible markets endpoint with caching and pagination
- `/v1/coins/list` - Supported coins list with platform information
- `/v1/asset_platforms` - CoinGecko-compatible asset platforms endpoint with 30-minute caching
- `/v1/coins/{coin_id}/market_chart` - Historical price data with intelligent caching
- `/v1/leaderboard/markets` - Top market data from leaderboard service
- `/v1/leaderboard/prices` - Top price data from leaderboard service  
- `/v1/leaderboard/simpleprices` - Simple prices for top tokens
- `/health` - Health check endpoint
- `/metrics` - Prometheus metrics


## Subprojects

### [Market Fetcher](./market-fetcher/README.md)

Go application that provides cached cryptocurrency data via REST API. The service fetches token lists, market data, and price updates from CoinGecko, and offers CoinGecko-compatible endpoints with intelligent caching. Also provides filtered token lists based on blockchain platforms.

### [Nginx Proxy](./nginx-proxy/README.md)

A proxy that performs caching, ETag optimization, and compression for displaying market data from market-fetcher.
