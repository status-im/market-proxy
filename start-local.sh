#!/bin/bash
# Stop and remove existing containers AND volumes
docker compose -f docker-compose.local.yml down --volumes

# Optional: remove unused volumes (for all projects)
docker volume prune -f

# Build and start containers
docker compose -f docker-compose.local.yml up --build -d

echo "Local environment started!"
echo "API proxy: http://localhost:8080"
echo "Frontend: http://localhost:3000"


