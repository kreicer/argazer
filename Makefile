# Makefile for Argazer

# Variables
APP_NAME := argazer
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DOCKER_IMAGE := $(APP_NAME):$(VERSION)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

.PHONY: all build clean install-hooks test lint deps run docker-build docker-push fmt security help

# Default target
all: clean deps build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(APP_NAME) .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(APP_NAME)

# Install git hooks
install-hooks:
	@echo "Installing git hooks..."
	@bash scripts/install-hooks.sh

# Run tests with coverage
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run --timeout=10m

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run the application locally
run:
	@echo "Running $(APP_NAME)..."
	./$(APP_NAME) --help

# Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .
	docker tag $(DOCKER_IMAGE) $(APP_NAME):latest

# Push Docker image
docker-push:
	@echo "Pushing Docker image $(DOCKER_IMAGE)..."
	docker push $(DOCKER_IMAGE)
	docker push $(APP_NAME):latest

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Security scan
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install it with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  install-hooks- Install git pre-push hooks"
	@echo "  test         - Run tests with coverage"
	@echo "  lint         - Lint code (timeout: 10m)"
	@echo "  deps         - Download dependencies"
	@echo "  run          - Run the application locally"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-push  - Push Docker image"
	@echo "  fmt          - Format code"
	@echo "  security     - Run security scan"
	@echo "  help         - Show this help message"

