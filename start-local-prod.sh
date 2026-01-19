#!/usr/bin/env bash

# This script starts the production configuration locally
# It sets MARKET_BASE_URL to the production URL to test the prod setup

echo "Starting local prod environment..."

if [ ! -d "./secrets" ]; then
    mkdir -p "./secrets"
fi

if [ ! -f "./secrets/.htpasswd" ]; then
    # Create .htpasswd with the user 'test' and password 'test'
    echo "test:$(openssl passwd -apr1 test)" > ./secrets/.htpasswd
fi

if [ ! -f "./secrets/coingecko_api_tokens.json" ]; then
    cat <<EOL > ./secrets/coingecko_api_tokens.json
{
  "api_tokens": [
    "Coingecko-token"
  ]
}
EOL
fi

# Create .env file with production MARKET_BASE_URL
# This emulates what Ansible does on production
echo "MARKET_BASE_URL=https://prod.market.status.im/" > .env

# Stop and remove existing containers AND volumes
docker compose -f docker-compose.yml down --volumes

# Optional: remove unused volumes (for all projects)
docker volume prune -f

# Build and start containers with production configuration
docker compose -f docker-compose.yml up --build -d

echo ""
echo "Local PROD environment started!"
echo "API proxy: http://localhost:8080"
echo "MARKET_BASE_URL set to: https://prod.market.status.im/"
echo ""
echo "Static files are available without authentication:"
echo "- Token lists: http://localhost:8080/static/lists.json"
echo "- Status token list: http://localhost:8080/static/status-token-list.json"
echo ""
echo "Check that lists.json contains prod URLs:"
echo "curl http://localhost:8080/static/lists.json | grep -o 'prod.market.status.im'"
echo ""
echo "Public API endpoints (no authentication required):"
echo "- Simple price (CoinGecko-compatible): http://localhost:8080/v1/simple/price"
echo "- Simple prices by token ID: http://localhost:8080/v1/leaderboard/simpleprices"
echo ""
echo "API endpoints (require authentication - user: test, password: test):"
echo "- Leaderboard prices: http://localhost:8080/v1/leaderboard/prices"
echo "- Leaderboard markets: http://localhost:8080/v1/leaderboard/markets"
echo "- Coins list: http://localhost:8080/v1/coins/list"
echo "- Coins markets (CoinGecko-compatible): http://localhost:8080/v1/coins/markets"

