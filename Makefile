.PHONY: help build test run-docker stop clean

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the router and cell binaries
	@echo "Building binaries..."
	@go build -o bin/router ./cmd/router
	@go build -o bin/cell ./cmd/cell
	@echo "Binaries built in ./bin/"

test: ## Run all tests
	@echo "Running tests..."
	@go test ./... -v

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

run-docker: ## Start all services with Docker Compose
	@echo "Starting services with Docker Compose..."
	docker compose up --build

stop: ## Stop Docker Compose services
	@echo "Stopping services..."
	docker compose down

logs: ## View router logs
	docker compose logs -f router

clean: ## Clean up build artifacts
	@echo "Cleaning up..."
	@rm -rf bin/ coverage.out coverage.html
	@go clean

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@golangci-lint run

tidy: ## Tidy Go modules
	@echo "Tidying modules..."
	@go mod tidy

.DEFAULT_GOAL := help
