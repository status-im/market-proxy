#!/usr/bin/env bash

# Set CORS origin for local development
export CORS_ORIGIN="http://localhost:3000"

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

if [ ! -f "./secrets/auth_config.json" ]; then
    # Puzzle auth config shared by the auth service and nginx
    cat <<EOL > ./secrets/auth_config.json
{
  "algorithm": "argon2id",
  "jwt_secret": "$(openssl rand -hex 32)",
  "puzzle_difficulty": 1,
  "requests_per_token": 100,
  "token_expiry_minutes": 10,
  "argon2_params": {
    "memory_kb": 16384,
    "time": 4,
    "threads": 4,
    "key_len": 32
  }
}
EOL
fi

# Stop and remove existing containers AND volumes
docker compose -f docker-compose.local.yml down --volumes

# Optional: remove unused volumes (for all projects)
docker volume prune -f

# Build and start containers
docker compose -f docker-compose.local.yml up --build -d

echo "Local environment started!"
echo "API proxy: http://localhost:8080"
echo "Frontend: http://localhost:3000"
echo "CORS origin set to: $CORS_ORIGIN"
echo ""
echo "Static files are available without authentication:"
echo "- Token lists: http://localhost:8080/static/token-lists.json"
echo "- Status token list: http://localhost:8080/static/status-token-list.json"
echo ""
echo "Public API endpoints (no authentication required):"
echo "- Simple price (CoinGecko-compatible): http://localhost:8080/v1/simple/price"
echo "- Simple prices by token ID: http://localhost:8080/v1/leaderboard/simpleprices"
echo ""
echo "API endpoints (require authentication - user: test, password: test,"
echo "or a puzzle JWT token from /auth/puzzle + /auth/solve):"
echo "- Leaderboard prices: http://localhost:8080/v1/leaderboard/prices"
echo "- Leaderboard markets: http://localhost:8080/v1/leaderboard/markets"
echo "- Coins list: http://localhost:8080/v1/coins/list"
echo "- Coins markets (CoinGecko-compatible): http://localhost:8080/v1/coins/markets"


