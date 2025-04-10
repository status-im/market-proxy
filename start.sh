#!/bin/bash

# Stop and remove existing containers AND volumes
docker compose -f docker-compose.yml down --volumes

# Optional: remove unused volumes (for all projects)
docker volume prune -f

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check if .htpasswd exists
if [ ! -f "./secrets/.htpasswd" ]; then
    echo "Error: ./secrets/.htpasswd file is missing"
    exit 1
fi

# Check if coingecko_api_tokens.json exists
if [ ! -f "./secrets/coingecko_api_tokens.json" ]; then
    echo "Error: ./secrets/coingecko_api_tokens.json file is missing"
    exit 1
fi

# Check if config.yaml exists
if [ ! -f "./market-fetcher/config.yaml" ]; then
    echo "Error: ./market-fetcher/config.yaml file is missing"
    exit 1
fi

# Build and start the containers
echo "Building and starting production environment..."
docker compose -f docker-compose.yml up --build -d

# Check if the containers are running
if [ $? -eq 0 ]; then
    echo "Production environment started!"
    echo "API proxy: http://localhost:8080"
    echo "Market fetcher: http://localhost:8081"
else
    echo "Error: Failed to start production environment"
    exit 1
fi 