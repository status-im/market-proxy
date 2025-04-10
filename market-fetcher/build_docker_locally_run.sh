#!/bin/bash

# Create Docker network (if it doesn't exist)
docker network create market-proxy-network || true

# Build image
docker build -t market-fetcher .

# Remove existing container (if it present)
docker rm -f market-fetcher || true

# Run container in network
docker run -d --name market-fetcher --network market-proxy-network -p 8080:8080  market-fetcher