# Local GitHub Actions Testing Guide

This guide shows you how to test GitHub Actions workflows locally before pushing to GitHub.

## 🔧 **Method 1: Using `act` (Recommended)**

`act` runs your GitHub Actions locally using Docker containers that simulate the GitHub Actions environment.

### Installation

```bash
# macOS
brew install act

# Linux
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Windows (using Chocolatey)
choco install act-cli
```

### Basic Usage

```bash
# List all available jobs
act --list

# Run all jobs (simulates push event)
act

# Run specific job
act -j test                    # Run unit tests
act -j integration-test        # Run integration tests
act -j lint                    # Run linting
act -j security               # Run security scan
act -j build                  # Run build

# Run with specific event
act push                      # Simulate push event
act pull_request             # Simulate PR event

# Run with specific architecture (for Apple M-series)
act --container-architecture linux/amd64

# Run with verbose output
act -v

# Dry run (show what would be executed)
act --dry-run
```

### Advanced Usage

```bash
# Use specific Docker image
act -P ubuntu-latest=catthehacker/ubuntu:act-latest

# Set environment variables
act --env GITHUB_TOKEN=your_token

# Use secrets file
act --secret-file .secrets

# Run specific workflow file
act -W .github/workflows/ci.yml

# Run with custom input
act workflow_dispatch --input key=value
```

## 🔧 **Method 2: Manual Local Testing**

You can also run the individual commands from your workflows manually:

### Unit Tests
```bash
# What the workflow does:
go test -v -race -coverprofile=coverage.out ./...
```

### Integration Tests
```bash
# What the workflow does:
docker-compose -f test/docker-compose.test.yml up -d --wait
cd test && go test -v -timeout=10m -tags=integration ./...
docker-compose -f test/docker-compose.test.yml down -v
```

### Linting
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linting
golangci-lint run --timeout=5m
```

### Security Scanning
```bash
# Install gosec
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Run security scan
gosec ./...
```

### Build
```bash
# Build for current platform
go build -o kim ./cmd/kim

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o kim-linux-amd64 ./cmd/kim
GOOS=darwin GOARCH=amd64 go build -o kim-darwin-amd64 ./cmd/kim
GOOS=windows GOARCH=amd64 go build -o kim-windows-amd64.exe ./cmd/kim
```

## 🔧 **Method 3: Using Makefile Targets**

We've already set up Makefile targets that mirror the CI workflow:

```bash
# Run what CI runs
make test                    # Unit tests
make test-integration        # Integration tests (requires Docker)
make lint                    # Linting (requires golangci-lint)
make build                   # Build binary
make build-all              # Multi-platform builds

# Run everything
make test-all               # Unit + integration tests
```

## 🐳 **Docker Requirements for `act`**

`act` requires Docker to be running. Make sure Docker Desktop is started:

```bash
# Check Docker status
docker version

# If Docker isn't running, start Docker Desktop
open -a Docker
```

## 📝 **Creating a Local Testing Script**

Let me create a script that runs all the checks locally:

```bash
#!/bin/bash
# local-ci.sh - Run CI checks locally

set -e

echo "🧪 Running local CI checks..."

echo "📋 1. Running unit tests..."
go test -v -race -coverprofile=coverage.out ./...

echo "🔍 2. Running linting..."
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run --timeout=5m
else
    echo "⚠️  golangci-lint not installed, skipping..."
fi

echo "🔒 3. Running security scan..."
if command -v gosec >/dev/null 2>&1; then
    gosec ./...
else
    echo "⚠️  gosec not installed, skipping..."
fi

echo "🔨 4. Building binary..."
go build -o kim ./cmd/kim

echo "🐳 5. Running integration tests..."
if command -v docker >/dev/null 2>&1 && command -v docker-compose >/dev/null 2>&1; then
    ./test/run-integration-tests.sh
else
    echo "⚠️  Docker not available, skipping integration tests..."
fi

echo "✅ All local CI checks completed!"
```

## 🚀 **Quick Start Examples**

### Test the Lint Job Locally
```bash
# Using act
act -j lint

# Using make
make lint

# Manual
golangci-lint run --timeout=5m
```

### Test the Security Job Locally
```bash
# Using act
act -j security

# Manual
gosec ./...
```

### Test Unit Tests Locally
```bash
# Using act
act -j test

# Using make
make test

# Manual
go test -v ./...
```

### Test Integration Tests Locally
```bash
# Using act (requires Docker in Docker, complex)
act -j integration-test

# Using make (recommended)
make test-integration

# Manual
./test/run-integration-tests.sh
```

## 🔧 **Troubleshooting**

### Common Issues with `act`

1. **Docker not running**:
   ```bash
   # Start Docker Desktop
   open -a Docker
   ```

2. **Apple M-series chip issues**:
   ```bash
   act --container-architecture linux/amd64
   ```

3. **Large Docker images**:
   ```bash
   # Use smaller images
   act -P ubuntu-latest=catthehacker/ubuntu:act-latest
   ```

4. **Missing secrets**:
   ```bash
   # Create .secrets file
   echo "GITHUB_TOKEN=your_token" > .secrets
   act --secret-file .secrets
   ```

### Performance Tips

1. **Use specific jobs**: `act -j test` instead of `act`
2. **Use smaller images**: `-P ubuntu-latest=catthehacker/ubuntu:act-latest`
3. **Cache Docker images**: Let act download images once
4. **Use make targets**: Often faster than full act simulation

## 📊 **Comparison of Methods**

| Method | Pros | Cons | Best For |
|--------|------|------|----------|
| `act` | Most accurate, full simulation | Requires Docker, slower | Final validation |
| Makefile | Fast, simple, integrated | Not exact CI environment | Daily development |
| Manual | Fastest, direct control | Manual steps, easy to miss | Quick checks |

## 🎯 **Recommended Workflow**

1. **During development**: Use `make test` and `make lint`
2. **Before committing**: Run `act -j test -j lint`
3. **Before pushing**: Run `act` (all jobs)
4. **For debugging CI**: Use `act -v` with specific job

This ensures your code will pass CI before you even push to GitHub!
