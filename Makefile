# Makefile for WebDAV Gateway

.PHONY: help build run stop clean test docker-build docker-up docker-down logs

help:
	@echo "WebDAV Gateway - Available commands:"
	@echo "  make build        - Build the Go binary"
	@echo "  make run          - Run the application locally"
	@echo "  make test         - Run tests"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-up    - Start all services with Docker Compose"
	@echo "  make docker-down  - Stop all services"
	@echo "  make logs         - View application logs"
	@echo "  make clean        - Clean build artifacts"

build:
	@echo "Building WebDAV Gateway..."
	go build -o bin/webdav-gateway ./cmd/server

run:
	@echo "Running WebDAV Gateway..."
	go run cmd/server/main.go cmd/server/auth_handlers.go cmd/server/share_handlers.go

test:
	@echo "Running tests..."
	go test -v -cover ./...

docker-build:
	@echo "Building Docker image..."
	docker build -t webdav-gateway:latest -f deployments/docker/Dockerfile .

docker-up:
	@echo "Starting services..."
	cd deployments/docker && docker-compose up -d
	@echo "Services started!"
	@echo "WebDAV Gateway: http://localhost:8080"
	@echo "MinIO Console: http://localhost:9001"

docker-down:
	@echo "Stopping services..."
	cd deployments/docker && docker-compose down

logs:
	cd deployments/docker && docker-compose logs -f webdav-gateway

clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

install:
	@echo "Installing dependencies..."
	go mod download

format:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linter..."
	golangci-lint run

dev:
	@echo "Running in development mode..."
	GIN_MODE=debug go run cmd/server/main.go cmd/server/auth_handlers.go cmd/server/share_handlers.go