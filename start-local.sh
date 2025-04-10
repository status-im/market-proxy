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


