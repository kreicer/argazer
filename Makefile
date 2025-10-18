# Makefile for Watcher

# Variables
APP_NAME := watcher
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DOCKER_IMAGE := $(APP_NAME):$(VERSION)
K8S_NAMESPACE := watcher

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

.PHONY: all build clean test deps run docker-build docker-push k8s-deploy k8s-clean help

# Default target
all: clean deps test build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(APP_NAME) .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(APP_NAME)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

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

# Deploy to Kubernetes
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -f k8s/
	@echo "Deployment complete. Don't forget to update the secret with your Telegram credentials:"
	@echo "kubectl create secret generic watcher-secrets \\"
	@echo "  --from-literal=telegram-webhook=\"https://api.telegram.org/bot<YOUR_BOT_TOKEN>/sendMessage\" \\"
	@echo "  --from-literal=telegram-chat-id=\"<YOUR_CHAT_ID>\" \\"
	@echo "  --namespace=$(K8S_NAMESPACE) \\"
	@echo "  --dry-run=client -o yaml | kubectl apply -f -"

# Clean Kubernetes resources
k8s-clean:
	@echo "Cleaning Kubernetes resources..."
	kubectl delete -f k8s/ --ignore-not-found=true

# Check Kubernetes deployment
k8s-status:
	@echo "Checking deployment status..."
	kubectl get pods -n $(K8S_NAMESPACE)
	kubectl logs -f deployment/$(APP_NAME) -n $(K8S_NAMESPACE)

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

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
	@echo "  test         - Run tests"
	@echo "  deps         - Download dependencies"
	@echo "  run          - Run the application locally"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-push  - Push Docker image"
	@echo "  k8s-deploy   - Deploy to Kubernetes"
	@echo "  k8s-clean    - Clean Kubernetes resources"
	@echo "  k8s-status   - Check Kubernetes deployment status"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  security     - Run security scan"
	@echo "  help         - Show this help message"

