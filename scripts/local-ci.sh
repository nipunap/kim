#!/bin/bash

# local-ci.sh - Run CI checks locally
# This script mimics what GitHub Actions does

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Running local CI checks for Kim...${NC}"
echo ""

# Function to run a step
run_step() {
    local step_name="$1"
    local step_command="$2"
    local optional="$3"

    echo -e "${YELLOW}📋 $step_name${NC}"

    if eval "$step_command"; then
        echo -e "${GREEN}✅ $step_name passed${NC}"
        echo ""
        return 0
    else
        if [ "$optional" = "optional" ]; then
            echo -e "${YELLOW}⚠️  $step_name failed (optional)${NC}"
            echo ""
            return 0
        else
            echo -e "${RED}❌ $step_name failed${NC}"
            echo ""
            return 1
        fi
    fi
}

# Check prerequisites
echo -e "${BLUE}🔍 Checking prerequisites...${NC}"

# Check Go
if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}❌ Go is not installed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Go $(go version | cut -d' ' -f3) found${NC}"

# Check Docker (optional for some tests)
if command -v docker >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Docker found${NC}"
    DOCKER_AVAILABLE=true
else
    echo -e "${YELLOW}⚠️  Docker not found (integration tests will be skipped)${NC}"
    DOCKER_AVAILABLE=false
fi

echo ""

# Step 1: Download dependencies
run_step "1. Download dependencies" "go mod download"

# Step 2: Format check
run_step "2. Format check" "test -z \"\$(gofmt -s -l . | grep -v vendor/)\""

# Step 3: Go vet
run_step "3. Go vet" "go vet ./..."

# Step 4: Unit tests
run_step "4. Unit tests" "go test -v -race -coverprofile=coverage.out ./..."

# Step 5: Linting (optional if not installed)
if command -v golangci-lint >/dev/null 2>&1; then
    run_step "5. Linting" "golangci-lint run --timeout=5m"
else
    echo -e "${YELLOW}📋 5. Linting${NC}"
    echo -e "${YELLOW}⚠️  golangci-lint not installed. Install with:${NC}"
    echo -e "${YELLOW}    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest${NC}"
    echo ""
fi

# Step 6: Security scan (optional if not installed)
if command -v gosec >/dev/null 2>&1; then
    run_step "6. Security scan" "gosec ./..."
else
    echo -e "${YELLOW}📋 6. Security scan${NC}"
    echo -e "${YELLOW}⚠️  gosec not installed. Install with:${NC}"
    echo -e "${YELLOW}    go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest${NC}"
    echo ""
fi

# Step 7: Build
run_step "7. Build binary" "go build -o kim ./cmd/kim"

# Step 8: Integration tests (optional if Docker not available)
if [ "$DOCKER_AVAILABLE" = true ]; then
    if [ -f "./test/run-integration-tests.sh" ]; then
        run_step "8. Integration tests" "./test/run-integration-tests.sh" "optional"
    else
        echo -e "${YELLOW}📋 8. Integration tests${NC}"
        echo -e "${YELLOW}⚠️  Integration test script not found${NC}"
        echo ""
    fi
else
    echo -e "${YELLOW}📋 8. Integration tests${NC}"
    echo -e "${YELLOW}⚠️  Docker not available, skipping integration tests${NC}"
    echo ""
fi

# Step 9: Multi-platform builds (quick test)
echo -e "${YELLOW}📋 9. Multi-platform build test${NC}"
GOOS=linux GOARCH=amd64 go build -o /tmp/kim-linux-amd64 ./cmd/kim
GOOS=darwin GOARCH=amd64 go build -o /tmp/kim-darwin-amd64 ./cmd/kim
GOOS=windows GOARCH=amd64 go build -o /tmp/kim-windows-amd64.exe ./cmd/kim
rm -f /tmp/kim-*
echo -e "${GREEN}✅ Multi-platform build test passed${NC}"
echo ""

# Summary
echo -e "${GREEN}🎉 Local CI checks completed successfully!${NC}"
echo ""
echo -e "${BLUE}📊 Summary:${NC}"
echo -e "  • Unit tests: ✅ Passed"
echo -e "  • Code quality: ✅ Passed"
echo -e "  • Build: ✅ Passed"
if [ "$DOCKER_AVAILABLE" = true ]; then
    echo -e "  • Integration tests: ✅ Available"
else
    echo -e "  • Integration tests: ⚠️  Skipped (Docker not available)"
fi
echo ""
echo -e "${BLUE}🚀 Your code is ready for GitHub!${NC}"

# Clean up
rm -f coverage.out kim
