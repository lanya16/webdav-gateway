#!/bin/bash

set -e

echo "Building WebDAV Gateway..."

# Build the Docker image
docker build -t webdav-gateway:latest -f deployments/docker/Dockerfile .

echo "Build completed successfully!"
echo "To start the services, run: cd deployments/docker && docker-compose up -d"