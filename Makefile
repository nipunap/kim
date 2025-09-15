# Kim - Kafka Management Tool Makefile

# Variables
BINARY_NAME=kim
MAIN_PATH=./cmd/kim
BUILD_DIR=./build
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test test-integration test-all deps fmt vet lint install uninstall help

# Default target
all: clean deps fmt vet test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: clean deps
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)

	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)

	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)

	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)

	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

	@echo "Built binaries for multiple platforms in $(BUILD_DIR)/"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@if command -v docker >/dev/null 2>&1 && command -v docker-compose >/dev/null 2>&1; then \
		./test/run-integration-tests.sh; \
	else \
		echo "Docker and Docker Compose are required for integration tests"; \
		echo "Install Docker: https://docs.docker.com/get-docker/"; \
		exit 1; \
	fi

# Run all tests (unit + integration)
test-all: test test-integration

# Start Kafka cluster for development
kafka-up:
	@echo "Starting Kafka cluster for development..."
	docker-compose -f test/docker-compose.test.yml up -d --wait
	@echo "Kafka cluster started at localhost:9092"
	@echo "Kafka UI available at http://localhost:8080 (run with --profile ui)"

# Stop Kafka cluster
kafka-down:
	@echo "Stopping Kafka cluster..."
	docker-compose -f test/docker-compose.test.yml down -v
	@echo "Kafka cluster stopped"

# Start Kafka cluster with UI
kafka-up-ui:
	@echo "Starting Kafka cluster with UI..."
	docker-compose -f test/docker-compose.test.yml --profile ui up -d --wait
	@echo "Kafka cluster started at localhost:9092"
	@echo "Kafka UI available at http://localhost:8080"

# Start SASL-enabled Kafka for authentication testing
kafka-up-sasl:
	@echo "Starting SASL-enabled Kafka cluster..."
	docker-compose -f test/docker-compose.test.yml --profile sasl up -d --wait
	@echo "SASL Kafka cluster started at localhost:9093"
	@echo "Use username: testuser, password: testpass"

# Show Kafka logs
kafka-logs:
	@echo "Showing Kafka logs..."
	docker-compose -f test/docker-compose.test.yml logs -f kafka

# Reset Kafka cluster (clean restart)
kafka-reset: kafka-down kafka-up

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Run golangci-lint (requires golangci-lint to be installed)
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(GOPATH)/bin/"

# Uninstall the binary from GOPATH/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME)"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

# Run in interactive mode
run-interactive: build
	@echo "Running $(BINARY_NAME) in interactive mode..."
	@$(BUILD_DIR)/$(BINARY_NAME) -i

# Development build (faster, no optimizations)
dev-build:
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Watch for changes and rebuild (requires entr: brew install entr)
watch:
	@echo "Watching for changes..."
	@find . -name "*.go" | entr -r make dev-build

# Generate mocks (requires mockgen: go install github.com/golang/mock/mockgen@latest)
mocks:
	@echo "Generating mocks..."
	@if command -v mockgen >/dev/null 2>&1; then \
		mockgen -source=internal/client/client.go -destination=internal/mocks/client_mock.go; \
		mockgen -source=internal/manager/topic.go -destination=internal/mocks/topic_mock.go; \
		mockgen -source=internal/manager/group.go -destination=internal/mocks/group_mock.go; \
		mockgen -source=internal/manager/message.go -destination=internal/mocks/message_mock.go; \
		echo "Mocks generated in internal/mocks/"; \
	else \
		echo "mockgen not installed. Install with: go install github.com/golang/mock/mockgen@latest"; \
	fi

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t kim:$(VERSION) .

# Docker run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -it kim:$(VERSION)

# Show help
help:
	@echo "Kim - Kafka Management Tool"
	@echo ""
	@echo "Available targets:"
	@echo "  build            Build the binary"
	@echo "  build-all        Build for multiple platforms"
	@echo "  clean            Clean build artifacts"
	@echo "  test             Run unit tests"
	@echo "  test-integration Run integration tests with Docker"
	@echo "  test-all         Run all tests (unit + integration)"
	@echo "  test-coverage    Run tests with coverage"
	@echo "  deps             Download dependencies"
	@echo "  fmt              Format code"
	@echo "  vet              Run go vet"
	@echo "  lint             Run golangci-lint"
	@echo "  install          Install binary to GOPATH/bin"
	@echo "  uninstall        Remove binary from GOPATH/bin"
	@echo "  run              Build and run the application"
	@echo "  run-interactive  Build and run in interactive mode"
	@echo "  dev-build        Fast development build"
	@echo "  watch            Watch for changes and rebuild"
	@echo "  mocks            Generate mocks"
	@echo "  docker-build     Build Docker image"
	@echo "  docker-run       Build and run Docker container"
	@echo "  kafka-up         Start Kafka cluster for development"
	@echo "  kafka-up-ui      Start Kafka cluster with UI"
	@echo "  kafka-up-sasl    Start SASL-enabled Kafka cluster"
	@echo "  kafka-down       Stop Kafka cluster"
	@echo "  kafka-logs       Show Kafka logs"
	@echo "  kafka-reset      Reset Kafka cluster (clean restart)"
	@echo "  help             Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build the binary"
	@echo "  make test                     # Run unit tests"
	@echo "  make test-integration         # Run integration tests"
	@echo "  make test-all                 # Run all tests"
	@echo "  make kafka-up                 # Start Kafka for development"
	@echo "  make run-interactive          # Run in interactive mode"
	@echo "  make build-all                # Build for all platforms"
