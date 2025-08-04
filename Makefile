# Makefile for Weather API
# A high-performance, multi-provider weather forecast API built with Go and Fiber

# Variables
BINARY_NAME=weather-api
BINARY_UNIX=$(BINARY_NAME)_unix
MAIN_PATH=./cmd/weather-api
BUILD_DIR=./build
DOCKER_IMAGE=weather-api
DOCKER_TAG=latest
CONFIG_FILE=./config/config.yaml

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run
GOLINT=$(shell go env GOPATH)/bin/golangci-lint
SWAG=$(shell go env GOPATH)/bin/swag

# Build flags
LDFLAGS=-ldflags "-w -s"
BUILD_FLAGS=-v

# Default target
.DEFAULT_GOAL := help

# Help target
.PHONY: help
help: ## Show this help message
	@echo "Weather API - Makefile Commands"
	@echo "================================"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
.PHONY: run
run: ## Run the application locally
	$(GORUN) $(MAIN_PATH)

.PHONY: run-dev
run-dev: ## Run the application in development mode with hot reload
	@echo "Starting development server..."
	$(GORUN) $(MAIN_PATH)

.PHONY: build
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

.PHONY: build-linux
build-linux: ## Build the application for Linux
	@echo "Building $(BINARY_NAME) for Linux..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) $(MAIN_PATH)

.PHONY: build-windows
build-windows: ## Build the application for Windows
	@echo "Building $(BINARY_NAME) for Windows..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)

.PHONY: build-mac
build-mac: ## Build the application for macOS
	@echo "Building $(BINARY_NAME) for macOS..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_mac $(MAIN_PATH)

.PHONY: build-all
build-all: ## Build the application for all platforms
	@echo "Building $(BINARY_NAME) for all platforms..."
	$(MAKE) build-linux
	$(MAKE) build-windows
	$(MAKE) build-mac

# Testing commands
.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	$(GOTEST) -race -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-benchmark
test-benchmark: ## Run benchmark tests
	@echo "Running benchmark tests..."
	$(GOTEST) -bench=. -benchmem ./...

# Code quality commands
.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	$(GOLINT) run

.PHONY: lint-fix
lint-fix: ## Run linter with auto-fix
	@echo "Running linter with auto-fix..."
	$(GOLINT) run --fix

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

.PHONY: tidy
tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	$(GOMOD) tidy

.PHONY: verify
verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GOMOD) verify

# Documentation commands
.PHONY: docs
docs: ## Generate Swagger documentation
	@echo "Generating Swagger documentation..."
	@if ! command -v $(SWAG) &> /dev/null; then \
		echo "swag is not installed. Installing..."; \
		$(GOGET) github.com/swaggo/swag/cmd/swag@latest; \
	fi
	$(SWAG) init -g $(MAIN_PATH)/main.go -o docs
	@echo "Swagger documentation generated successfully!"

.PHONY: docs-serve
docs-serve: ## Serve Swagger documentation
	@echo "Serving Swagger documentation at http://localhost:8080/swagger/"
	$(GORUN) $(MAIN_PATH)

# Dependency management
.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Docker commands
.PHONY: docker-build
docker-build: ## Build Docker image (production - scratch)
	@echo "Building production Docker image (scratch)..."
	docker build --target runtime -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f ./deployment/docker/Dockerfile .

.PHONY: docker-build-alpine
docker-build-alpine: ## Build Docker image (development - alpine)
	@echo "Building development Docker image (alpine)..."
	docker build --target runtime-alpine -t $(DOCKER_IMAGE):$(DOCKER_TAG)-alpine -f ./deployment/docker/Dockerfile .

.PHONY: docker-build-dev
docker-build-dev: ## Build Docker image (development - with source)
	@echo "Building development Docker image (with source)..."
	docker build --target builder -t $(DOCKER_IMAGE):$(DOCKER_TAG)-dev -f ./deployment/docker/Dockerfile .

.PHONY: docker-run
docker-run: ## Run Docker container (production)
	@echo "Running production Docker container..."
	docker run -p 8080:8080 --name $(BINARY_NAME) $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-run-alpine
docker-run-alpine: ## Run Docker container (alpine)
	@echo "Running Alpine Docker container..."
	docker run -p 8080:8080 --name $(BINARY_NAME)-alpine $(DOCKER_IMAGE):$(DOCKER_TAG)-alpine

.PHONY: docker-run-dev
docker-run-dev: ## Run Docker container (development)
	@echo "Running development Docker container..."
	docker run -p 8080:8080 -v $(PWD):/app --name $(BINARY_NAME)-dev $(DOCKER_IMAGE):$(DOCKER_TAG)-dev

.PHONY: docker-compose-up
docker-compose-up: ## Start services with Docker Compose
	@echo "Starting services with Docker Compose..."
	docker-compose -f ./deployment/docker/docker-compose.yml up -d

.PHONY: docker-compose-down
docker-compose-down: ## Stop services with Docker Compose
	@echo "Stopping services with Docker Compose..."
	docker-compose -f ./deployment/docker/docker-compose.yml down

.PHONY: docker-compose-dev
docker-compose-dev: ## Start development environment
	@echo "Starting development environment..."
	docker-compose -f ./deployment/docker/docker-compose.yml --profile dev up -d

.PHONY: docker-stop
docker-stop: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	docker stop $(BINARY_NAME) $(BINARY_NAME)-alpine $(BINARY_NAME)-dev || true
	docker rm $(BINARY_NAME) $(BINARY_NAME)-alpine $(BINARY_NAME)-dev || true

.PHONY: docker-clean
docker-clean: ## Clean Docker images
	@echo "Cleaning Docker images..."
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):$(DOCKER_TAG)-alpine $(DOCKER_IMAGE):$(DOCKER_TAG)-dev || true

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-scan
docker-scan: ## Security scan Docker image
	@echo "Scanning Docker image for vulnerabilities..."
	docker scan $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-multiarch
docker-multiarch: ## Build multi-architecture Docker image
	@echo "Building multi-architecture Docker image..."
	docker buildx build --platform linux/amd64,linux/arm64 --target runtime -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f ./deployment/docker/Dockerfile .

# Cleanup commands
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

.PHONY: clean-all
clean-all: clean docker-clean ## Clean everything including Docker images

# Installation commands
.PHONY: install
install: build ## Install the application
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	$(GOGET) github.com/swaggo/swag/cmd/swag@latest
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed. Make sure $(GOPATH)/bin is in your PATH"

# Security commands
.PHONY: security-check
security-check: ## Run security checks
	@echo "Running security checks..."
	$(GOCMD) list -json -deps ./... | nancy sleuth

.PHONY: audit
audit: ## Audit dependencies
	@echo "Auditing dependencies..."
	$(GOMOD) download
	$(GOCMD) list -json -deps ./... | nancy sleuth

# Development workflow
.PHONY: dev-setup
dev-setup: install-tools deps docs ## Setup development environment
	@echo "Development environment setup complete!"

.PHONY: pre-commit
pre-commit: fmt lint test ## Run pre-commit checks
	@echo "Pre-commit checks completed!"

.PHONY: ci
ci: deps lint test-coverage build ## Run CI pipeline
	@echo "CI pipeline completed!"

# Monitoring and debugging
.PHONY: profile
profile: ## Generate CPU profile
	@echo "Generating CPU profile..."
	$(GOCMD) run -cpuprofile=cpu.prof $(MAIN_PATH)

.PHONY: profile-memory
profile-memory: ## Generate memory profile
	@echo "Generating memory profile..."
	$(GOCMD) run -memprofile=mem.prof $(MAIN_PATH)

# Database and migration commands (if needed in the future)
.PHONY: migrate
migrate: ## Run database migrations
	@echo "Running database migrations..."
	@echo "No migrations configured yet"

.PHONY: seed
seed: ## Seed database with test data
	@echo "Seeding database..."
	@echo "No seeding configured yet"

# Utility commands
.PHONY: version
version: ## Show version information
	@echo "Weather API Version Information"
	@echo "================================"
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Build time: $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')"

.PHONY: check
check: ## Check if all required tools are installed
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed. Aborting." >&2; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required but not installed. Aborting." >&2; exit 1; }
	@echo "All required tools are installed!"

 