.PHONY: help build test migrate-up migrate-down migrate-create run-api run-worker docker-up docker-down clean lint fmt

# Default target
help:
	@echo "Available targets:"
	@echo "  build          - Build API and worker binaries"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  migrate-up     - Run database migrations up"
	@echo "  migrate-down   - Run database migrations down"
	@echo "  migrate-create - Create new migration (usage: make migrate-create NAME=migration_name)"
	@echo "  run-api        - Run API server"
	@echo "  run-worker     - Run worker"
	@echo "  docker-up      - Start all services with docker-compose"
	@echo "  docker-down    - Stop all docker-compose services"
	@echo "  clean          - Remove build artifacts"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  deps           - Download dependencies"

# Build targets
build:
	@echo "Building API server..."
	@go build -o bin/api cmd/api/main.go
	@echo "Building worker..."
	@go build -o bin/worker cmd/worker/main.go
	@echo "Build complete!"

# Test targets
test:
	@echo "Running tests..."
	@go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

test-unit:
	@echo "Running unit tests..."
	@go test -v ./tests/unit/...

test-integration:
	@echo "Running integration tests..."
	@go test -v ./tests/integration/...

test-contract:
	@echo "Running contract tests..."
	@go test -v ./tests/contract/...

# Database migration targets
migrate-up:
	@echo "Running migrations up..."
	@migrate -path ./migrations -database "$${DATABASE_URL}" up

migrate-down:
	@echo "Rolling back last migration..."
	@migrate -path ./migrations -database "$${DATABASE_URL}" down 1

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: Please provide a migration name. Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration: $(NAME)"
	@migrate create -ext sql -dir migrations -seq $(NAME)

# Run targets
run-api:
	@echo "Starting API server..."
	@go run cmd/api/main.go

run-worker:
	@echo "Starting worker..."
	@go run cmd/worker/main.go

# Docker targets
docker-up:
	@echo "Starting services with docker-compose..."
	@docker-compose up -d
	@echo "Services started! API available at http://localhost:8080"

docker-down:
	@echo "Stopping docker-compose services..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

# Utility targets
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete!"

lint:
	@echo "Running linter..."
	@golangci-lint run

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete!"

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated!"

# Development helpers
dev-api: deps
	@echo "Starting API in development mode..."
	@air -c .air.toml || go run cmd/api/main.go

dev-worker: deps
	@echo "Starting worker in development mode..."
	@go run cmd/worker/main.go

# Database helpers
db-reset: migrate-down migrate-up
	@echo "Database reset complete!"

db-status:
	@migrate -path ./migrations -database "$${DATABASE_URL}" version

# Install tools
install-tools:
	@echo "Installing development tools..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Tools installed!"
