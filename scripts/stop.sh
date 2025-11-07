#!/bin/bash

set -e

echo "Stopping WebDAV Gateway services..."

cd deployments/docker

# Stop all services
docker-compose down

echo "Services stopped successfully!"