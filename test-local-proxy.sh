#!/bin/bash

# Check if .env exists
if [ ! -f ".env" ]; then
    echo "Error: .env file is missing"
    exit 1
fi

# Source the .env file to get credentials
source .env

# Check if required environment variables are set
if [ -z "$MARKET_PROXY_USER" ] || [ -z "$MARKET_PROXY_PASSWORD" ]; then
    echo "Error: MARKET_PROXY_USER and MARKET_PROXY_PASSWORD must be set in .env file"
    exit 1
fi

# Make the request to the local proxy
curl -s -u "$MARKET_PROXY_USER:$MARKET_PROXY_PASSWORD" http://localhost:8080/v1/leaderboard/markets | jq '.["data"][0]'
