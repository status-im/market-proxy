# CoinGecko Package

This package implements a client for fetching cryptocurrency data from the CoinGecko API.

## Package Structure

```mermaid
graph TD
    subgraph "CoinGecko Package"
        Service[Service]
        Config[Config]
        Cache[API Response Cache]
        PaginatedFetcher[PaginatedFetcher]
        CoinGeckoClient[CoinGeckoClient]
        HTTPClientWithRetries[HTTPClientWithRetries]
        APIKeyManager[API Key Manager]
    end

    Service --> Config
    Service --> Cache
    Service --> PaginatedFetcher
    
    PaginatedFetcher --> CoinGeckoClient
    PaginatedFetcher -.-> |"Pages API requests"| CoinGeckoClient
    
    CoinGeckoClient --> HTTPClientWithRetries
    CoinGeckoClient --> APIKeyManager
    
    APIKeyManager -.-> |"Rotates API keys"| HTTPClientWithRetries
    
    style Service fill:#f9f,stroke:#333,stroke-width:2px
    style PaginatedFetcher fill:#bbf,stroke:#333,stroke-width:2px
    style CoinGeckoClient fill:#bbf,stroke:#333,stroke-width:2px
    style HTTPClientWithRetries fill:#dfd,stroke:#333,stroke-width:2px
    style APIKeyManager fill:#dfd,stroke:#333,stroke-width:2px
```

## Core Components

- **Service**: Main entry point that receives configuration and tokens, maintains a cache of API responses
- **PaginatedFetcher**: Handles pagination of large requests by breaking them into smaller page requests
- **CoinGeckoClient**: Manages the communication with the CoinGecko API
- **HTTPClientWithRetries**: Implements retry logic for API requests
- **API Key Manager**: Rotates through available API keys to avoid rate limiting 