# Testing Guide for Kim

This document describes the testing strategy and how to run tests for the Kim Kafka management tool.

## Test Structure

Kim has a comprehensive test suite with multiple layers:

### 1. Unit Tests
- **Location**: `internal/*/test.go` files
- **Purpose**: Test individual components in isolation
- **Dependencies**: Mock implementations, no external services
- **Speed**: Fast (< 1 second per package)

### 2. Integration Tests
- **Location**: `test/integration_test.go`
- **Purpose**: Test Kim against real Kafka clusters
- **Dependencies**: Docker, Docker Compose, real Kafka cluster
- **Speed**: Slower (2-5 minutes)

### 3. Mock Framework
- **Location**: `internal/testutil/mocks.go`
- **Purpose**: Provide mock implementations for testing
- **Features**: Mock Kafka clients, consumers, producers

## Running Tests

### Prerequisites

For unit tests:
```bash
go version  # Go 1.21 or later
```

For integration tests:
```bash
docker --version        # Docker 20.10 or later
docker-compose --version # Docker Compose 2.0 or later
```

### Unit Tests

Run all unit tests:
```bash
make test
```

Run tests with coverage:
```bash
make test-coverage
open coverage.html  # View coverage report
```

Run specific package tests:
```bash
go test -v ./internal/config
go test -v ./internal/auth
go test -v ./internal/logger
```

### Integration Tests

Run integration tests (requires Docker):
```bash
make test-integration
```

Or run the integration test script directly:
```bash
./test/run-integration-tests.sh
```

Run all tests (unit + integration):
```bash
make test-all
```

### Development Testing

Start Kafka cluster for manual testing:
```bash
make kafka-up
```

Start Kafka with UI for visual inspection:
```bash
make kafka-up-ui
# Open http://localhost:8080 in browser
```

Start SASL-enabled Kafka for authentication testing:
```bash
make kafka-up-sasl
# Use username: testuser, password: testpass
```

Stop Kafka cluster:
```bash
make kafka-down
```

## Integration Test Details

### Test Scenarios

The integration tests cover:

1. **Profile Management**
   - Adding Kafka profiles
   - Listing profiles
   - Setting active profiles
   - Profile validation

2. **Topic Operations**
   - Creating topics with various configurations
   - Listing topics with pagination and filtering
   - Describing topic details
   - Deleting topics

3. **Message Operations**
   - Producing messages with keys, values, and headers
   - Consuming messages from beginning
   - Batch message operations
   - Consumer group creation

4. **Consumer Group Management**
   - Listing consumer groups
   - Describing group details
   - Resetting offsets
   - Deleting groups

5. **Output Formats**
   - Table format (default)
   - JSON format
   - YAML format

6. **Error Handling**
   - Invalid parameters
   - Non-existent resources
   - Connection failures

### Test Environment

The integration tests use:
- **Kafka**: Confluent Platform 7.4.0
- **Zookeeper**: Confluent Platform 7.4.0
- **Network**: Isolated Docker network
- **Ports**:
  - Kafka: 9092 (PLAINTEXT)
  - Kafka SASL: 9093 (SASL_PLAINTEXT)
  - Zookeeper: 2181
  - Kafka UI: 8080 (optional)

### Test Data Cleanup

Integration tests automatically:
- Create unique topic/group names with timestamps
- Clean up created resources after tests
- Stop and remove Docker containers
- Remove temporary configuration files

## Continuous Integration

### GitHub Actions

The CI pipeline runs:
1. **Unit Tests**: Fast feedback on code changes
2. **Integration Tests**: Full end-to-end testing
3. **Build**: Multi-platform binary builds
4. **Lint**: Code quality checks
5. **Security**: Vulnerability scanning

### Test Matrix

CI tests against:
- **Go versions**: 1.21
- **Operating systems**: Ubuntu (Linux)
- **Architectures**: amd64, arm64
- **Kafka versions**: 7.4.0

## Writing Tests

### Unit Test Guidelines

```go
func TestMyFunction(t *testing.T) {
    // Arrange
    mockClient := testutil.NewMockClient(testutil.TestProfile(), testutil.TestLogger())

    // Act
    result, err := MyFunction(mockClient)

    // Assert
    testutil.AssertNoError(t, err)
    testutil.AssertEqual(t, expectedResult, result)
}
```

### Integration Test Guidelines

```go
func TestIntegrationMyFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    setupTestProfile(t)

    // Test against real Kafka
    output, err := runKimCommand("my-command", "arg1", "arg2")

    if err != nil {
        t.Fatalf("Command failed: %v\nOutput: %s", err, output)
    }

    if !strings.Contains(output, "expected-string") {
        t.Errorf("Expected output to contain 'expected-string', got: %s", output)
    }
}
```

### Mock Guidelines

Use mocks for:
- External dependencies (Kafka, AWS)
- Network operations
- File system operations
- Time-dependent operations

Don't mock:
- Simple data structures
- Pure functions
- Internal business logic

## Test Performance

### Benchmarks

Run benchmarks:
```bash
go test -bench=. -benchmem ./...
```

### Test Timing

Typical test execution times:
- Unit tests: ~2-5 seconds
- Integration tests: ~2-5 minutes
- Full CI pipeline: ~8-12 minutes

### Optimization Tips

1. **Parallel Tests**: Use `t.Parallel()` for independent tests
2. **Test Caching**: Reuse test fixtures when possible
3. **Selective Testing**: Use build tags for expensive tests
4. **Mock Efficiency**: Prefer mocks over real services in unit tests

## Troubleshooting

### Common Issues

**Docker not available**:
```bash
# Install Docker Desktop or Docker Engine
# Ensure Docker daemon is running
docker version
```

**Port conflicts**:
```bash
# Check if ports are in use
lsof -i :9092
lsof -i :2181

# Stop conflicting services
make kafka-down
```

**Test timeouts**:
```bash
# Increase timeout for slow systems
go test -timeout=15m ./test/...
```

**Memory issues**:
```bash
# Increase Docker memory limits
# Docker Desktop > Settings > Resources > Memory
```

### Debug Commands

Show Kafka logs:
```bash
make kafka-logs
```

Check Kafka health:
```bash
docker-compose -f docker-compose.test.yml exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092
```

List topics manually:
```bash
docker-compose -f docker-compose.test.yml exec kafka kafka-topics --bootstrap-server localhost:9092 --list
```

## Contributing

When contributing tests:

1. **Add unit tests** for all new functionality
2. **Add integration tests** for user-facing features
3. **Update mocks** when changing interfaces
4. **Run full test suite** before submitting PRs
5. **Document test scenarios** in commit messages

### Test Coverage Goals

- **Unit tests**: > 80% coverage
- **Integration tests**: Cover all CLI commands
- **Critical paths**: 100% coverage for auth, config, core operations

## Resources

- [Kim Repository](https://github.com/nipunap/kim)
- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Framework](https://github.com/stretchr/testify)
- [Docker Compose](https://docs.docker.com/compose/)
- [Confluent Platform](https://docs.confluent.io/platform/current/overview.html)
