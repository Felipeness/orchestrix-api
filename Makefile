.PHONY: all build run test clean generate generate-db generate-api lint help

# Variables
BINARY_NAME=orchestrix-api
WORKER_BINARY=orchestrix-worker
OPENAPI_SPEC=../api/openapi/v1/openapi.bundled.yaml

# Default target
all: generate build

# Build the API binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) ./cmd/api

# Build the worker binary
build-worker:
	@echo "Building $(WORKER_BINARY)..."
	go build -o bin/$(WORKER_BINARY) ./cmd/worker

# Run the API server
run:
	go run ./cmd/api

# Run the worker
run-worker:
	go run ./cmd/worker

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf internal/api/oas/

# Generate all code
generate: generate-db generate-api

# Generate database code with sqlc
generate-db:
	@echo "Generating database code with sqlc..."
	sqlc generate

# Generate API code with ogen
generate-api:
	@echo "Bundling OpenAPI spec..."
	cd .. && bunx @redocly/cli bundle api/openapi/v1/openapi.yaml -o api/openapi/v1/openapi.bundled.yaml
	@echo "Generating API code with ogen..."
	go run github.com/ogen-go/ogen/cmd/ogen@latest \
		--target ./internal/api/oas \
		--package oas \
		--clean \
		$(OPENAPI_SPEC)

# Install development tools
tools:
	@echo "Installing development tools..."
	go install github.com/ogen-go/ogen/cmd/ogen@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Lint the code
lint:
	@echo "Running linter..."
	golangci-lint run

# Format the code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Help
help:
	@echo "Available targets:"
	@echo "  all          - Generate code and build"
	@echo "  build        - Build the API binary"
	@echo "  build-worker - Build the worker binary"
	@echo "  run          - Run the API server"
	@echo "  run-worker   - Run the worker"
	@echo "  test         - Run tests"
	@echo "  clean        - Remove build artifacts"
	@echo "  generate     - Generate all code (db + api)"
	@echo "  generate-db  - Generate database code with sqlc"
	@echo "  generate-api - Generate API code with ogen"
	@echo "  tools        - Install development tools"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  tidy         - Tidy dependencies"
	@echo "  help         - Show this help"
