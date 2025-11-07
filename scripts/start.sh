#!/bin/bash

set -e

echo "Starting WebDAV Gateway services..."

cd deployments/docker

# Start all services
docker-compose up -d

echo "Services started successfully!"
echo ""
echo "Service URLs:"
echo "  WebDAV Gateway: http://localhost:8080"
echo "  MinIO Console:  http://localhost:9001 (admin/minioadmin)"
echo ""
echo "To view logs: docker-compose logs -f webdav-gateway"
echo "To stop: docker-compose down"