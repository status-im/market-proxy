#!/bin/bash

# Create Docker network (if it doesn't exist)
docker network create market-proxy-network || true

# Build image. The build context is the repository root: the image needs the
# proxy-common submodule next to market-fetcher.
cd "$(dirname "$0")/.."
docker build -f market-fetcher/Dockerfile -t market-fetcher .

# Remove existing container (if it present)
docker rm -f market-fetcher || true

# Run container in network
docker run -d --name market-fetcher --network market-proxy-network -p 8080:8080  market-fetcher