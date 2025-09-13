#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔍 Checking Docker setup for Kim integration tests...${NC}"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    echo "Please install Docker Desktop from: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    echo "Please install Docker Compose from: https://docs.docker.com/compose/install/"
    exit 1
fi

echo -e "${GREEN}✅ Docker and Docker Compose are installed${NC}"

# Check if Docker daemon is running
if ! docker info &> /dev/null; then
    echo -e "${RED}❌ Docker daemon is not running${NC}"
    echo "Please start Docker Desktop or the Docker daemon"
    echo ""
    echo "On macOS/Windows: Start Docker Desktop application"
    echo "On Linux: sudo systemctl start docker"
    exit 1
fi

echo -e "${GREEN}✅ Docker daemon is running${NC}"

# Test Docker functionality
if ! docker run --rm hello-world &> /dev/null; then
    echo -e "${RED}❌ Docker is not working properly${NC}"
    echo "Please check Docker installation and permissions"
    exit 1
fi

echo -e "${GREEN}✅ Docker is working properly${NC}"

# Check available disk space (Docker needs space for images)
AVAILABLE_SPACE=$(df -h . | awk 'NR==2 {print $4}' | sed 's/G//')
if [ "${AVAILABLE_SPACE%.*}" -lt 2 ]; then
    echo -e "${YELLOW}⚠️  Warning: Low disk space (${AVAILABLE_SPACE}G available)${NC}"
    echo "Docker images require ~1-2GB of space"
fi

echo -e "${GREEN}✅ Docker setup is ready for Kim integration tests!${NC}"
echo ""
echo "You can now run:"
echo "  make kafka-up           # Start Kafka cluster"
echo "  make test-integration   # Run integration tests"
echo "  make kafka-down         # Stop Kafka cluster"
