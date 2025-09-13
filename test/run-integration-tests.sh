rfectly sh#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DOCKER_COMPOSE_FILE="docker-compose.test.yml"
TEST_TIMEOUT="10m"

echo -e "${YELLOW}🚀 Starting Kim Integration Tests${NC}"

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed or not in PATH${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed or not in PATH${NC}"
    exit 1
fi

# Function to cleanup
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up Docker containers...${NC}"
    docker-compose -f "$DOCKER_COMPOSE_FILE" down -v --remove-orphans 2>/dev/null || true

    # Remove any leftover kim binary
    rm -f ./kim

    echo -e "${GREEN}✅ Cleanup completed${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Build Kim binary
echo -e "${YELLOW}🔨 Building Kim binary...${NC}"
go build -o kim ./cmd/kim
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to build Kim binary${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Kim binary built successfully${NC}"

# Start Kafka cluster
echo -e "${YELLOW}🐳 Starting Kafka cluster...${NC}"
docker-compose -f "$DOCKER_COMPOSE_FILE" up -d --wait
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to start Kafka cluster${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Kafka cluster started successfully${NC}"

# Wait a bit more for Kafka to be fully ready
echo -e "${YELLOW}⏳ Waiting for Kafka to be fully ready...${NC}"
sleep 10

# Check Kafka health
echo -e "${YELLOW}🔍 Checking Kafka health...${NC}"
docker-compose -f "$DOCKER_COMPOSE_FILE" exec -T kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Kafka is not healthy${NC}"
    docker-compose -f "$DOCKER_COMPOSE_FILE" logs kafka
    exit 1
fi
echo -e "${GREEN}✅ Kafka is healthy${NC}"

# Run integration tests
echo -e "${YELLOW}🧪 Running integration tests...${NC}"
cd test
go test -v -timeout="$TEST_TIMEOUT" -tags=integration ./...
TEST_RESULT=$?

if [ $TEST_RESULT -eq 0 ]; then
    echo -e "${GREEN}✅ All integration tests passed!${NC}"
else
    echo -e "${RED}❌ Some integration tests failed${NC}"

    # Show Kafka logs for debugging
    echo -e "${YELLOW}📋 Kafka logs for debugging:${NC}"
    docker-compose -f "../$DOCKER_COMPOSE_FILE" logs --tail=50 kafka
fi

exit $TEST_RESULT
