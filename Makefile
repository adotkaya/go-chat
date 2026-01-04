.PHONY: build run migrate-up migrate-down test clean

# Build the application
build:
	@echo "Building application..."
	@go build -o bin/chat cmd/api/*.go

# Run the application
run:
	@go run cmd/api/*.go

# Run migrations (up)
migrate-up:
	@echo "Running migrations..."
	@go run cmd/migrate/main.go up

# Rollback last migration (down)
migrate-down:
	@echo "Rolling back last migration..."
	@go run cmd/migrate/main.go down

# Run tests
test:
	@go test -v ./...

# Clean build artifacts
clean:
	@rm -rf bin/
	@echo "Cleaned build artifacts"

# Install dependencies
deps:
	@go mod tidy
	@go mod download

# Development setup
setup: deps migrate-up
	@echo "Development environment ready!"
