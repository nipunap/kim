# Kim - Kafka Management Tool

[![Go Version](https://img.shields.io/github/go-mod/go-version/nipunap/kim)](https://golang.org/)
[![License](https://img.shields.io/github/license/nipunap/kim)](https://github.com/nipunap/kim/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/release/nipunap/kim.svg)](https://github.com/nipunap/kim/releases)
[![CI](https://github.com/nipunap/kim/workflows/CI/badge.svg)](https://github.com/nipunap/kim/actions)

Kim is a powerful command-line interface for managing Kafka and MSK clusters, written in Go. It provides an intuitive way to interact with Kafka topics, consumer groups, and messages with support for both regular Kafka and AWS MSK clusters.

Inspired by the Python Kombucha tool, Kim offers the same functionality with improved performance, better structure, and enhanced user experience.

## Features

- **Profile Management**: Support for multiple Kafka/MSK cluster configurations
- **Topic Management**: List, describe, create, and delete Kafka topics
- **Consumer Group Management**: Monitor and manage consumer groups with lag information
- **Message Operations**: Produce and consume messages with real-time streaming
- **Interactive Mode**: Vim-like navigation with live updates and search functionality
- **Authentication**: Support for MSK IAM, SSL, and SASL authentication
- **Multiple Output Formats**: Table, JSON, and YAML output formats
- **Structured Logging**: Comprehensive logging with configurable levels

## Installation

### From Source

```bash
git clone https://github.com/nipunap/kim.git
cd kim
make build
# Binary will be available at ./build/kim
```

### Using Go

```bash
go install github.com/nipunap/kim/cmd/kim@latest
```

### Pre-built Binaries

Download pre-built binaries from the [releases page](https://github.com/nipunap/kim/releases) for your platform.

## Quick Start

1. **Add a Kafka profile:**
   ```bash
   kim profile add local --type kafka --bootstrap-servers localhost:9092
   kim profile use local
   ```

2. **List topics:**
   ```bash
   kim topic list
   ```

3. **Start interactive mode:**
   ```bash
   kim -i
   ```

## Usage

### Profile Management

Kim uses profiles to manage connections to different Kafka clusters.

```bash
# Add a local Kafka profile
kim profile add local --type kafka --bootstrap-servers localhost:9092

# Add an MSK profile with IAM authentication
kim profile add prod-msk --type msk --region us-east-1 \
  --cluster-arn "arn:aws:kafka:us-east-1:123456789012:cluster/my-cluster/uuid"

# Add a Kafka profile with SSL
kim profile add secure-kafka --type kafka \
  --bootstrap-servers kafka.example.com:9093 \
  --security-protocol SSL \
  --ssl-ca-file /path/to/ca.pem

# Add a Kafka profile with SASL authentication
kim profile add sasl-kafka --type kafka \
  --bootstrap-servers kafka.example.com:9093 \
  --security-protocol SASL_SSL \
  --sasl-mechanism PLAIN \
  --sasl-username myuser \
  --sasl-password mypass

# List all profiles
kim profile list

# Switch to a profile
kim profile use prod-msk

# Delete a profile
kim profile delete old-profile
```

### Topic Management

```bash
# List all topics
kim topic list

# List topics with pagination
kim topic list --page 2 --page-size 10

# List topics with pattern filtering
kim topic list --pattern "user-*"

# Describe a specific topic
kim topic describe my-topic

# Create a new topic
kim topic create my-new-topic --partitions 3 --replication-factor 2

# Create a topic with custom configuration
kim topic create my-topic --partitions 6 --replication-factor 3 \
  --config retention.ms=604800000 \
  --config cleanup.policy=delete

# Delete a topic
kim topic delete my-old-topic

# Delete a topic without confirmation
kim topic delete my-old-topic --force
```

### Consumer Group Management

```bash
# List all consumer groups
kim group list

# List groups with pattern filtering
kim group list --pattern "app-*"

# Describe a specific consumer group
kim group describe my-consumer-group

# Delete a consumer group
kim group delete old-group

# Reset consumer group offsets to earliest
kim group reset my-group --to-earliest

# Reset consumer group offsets to latest
kim group reset my-group --to-latest

# Reset consumer group offsets to specific offset
kim group reset my-group --to-offset 1000
```

### Message Operations

```bash
# Produce a simple message
kim message produce my-topic --value "Hello, World!"

# Produce a message with key
kim message produce my-topic --key "user123" --value "User data"

# Produce a message to specific partition
kim message produce my-topic --value "Message" --partition 2

# Produce a message with headers
kim message produce my-topic --value "Message" \
  --header "source=app1" --header "version=1.0"

# Consume messages from beginning
kim message consume my-topic --group-id my-consumer --from-beginning

# Consume messages with timeout
kim message consume my-topic --group-id my-consumer --timeout 30s

# Consume limited number of messages
kim message consume my-topic --group-id my-consumer --max-messages 100
```

### Interactive Mode

Kim provides a powerful interactive mode with vim-like navigation:

```bash
kim -i
```

**Interactive Mode Commands:**
- `:help` - Show help
- `:topics` - List all topics
- `:topic describe <name>` - Describe a topic
- `:groups` - List consumer groups
- `:group describe <id>` - Describe a consumer group
- `:profile list` - List profiles
- `:profile use <name>` - Switch profile
- `:message consume <topic>` - Start consuming messages
- `:message stop` - Stop consuming messages
- `:q` or `:quit` - Exit

**Navigation:**
- `j/k` - Scroll down/up
- `f/b` - Page down/up
- `g/G` - Go to top/bottom
- `/<pattern>` - Search
- `n/p` - Next/previous search result
- `r` - Refresh current view

### Output Formats

Kim supports multiple output formats:

```bash
# Table format (default)
kim topic list

# JSON format
kim topic list --format json

# YAML format
kim topic describe my-topic --format yaml
```

### Debug Mode

Enable debug logging for troubleshooting:

```bash
kim --debug topic list
```

## Configuration

Kim stores configuration in `~/.kim/config.yaml`. The configuration file is automatically created on first run.

Example configuration:
```yaml
profiles:
  local:
    name: local
    type: kafka
    bootstrap_servers: localhost:9092
    security_protocol: PLAINTEXT
  prod-msk:
    name: prod-msk
    type: msk
    region: us-east-1
    cluster_arn: arn:aws:kafka:us-east-1:123456789012:cluster/my-cluster/uuid
    auth_method: IAM
active_profile: local
settings:
  page_size: 20
  refresh_interval: 10
  default_format: table
  color_scheme: default
  vim_mode: true
```

## Architecture

Kim follows a clean architecture pattern with clear separation of concerns:

```
kim/
├── cmd/kim/                 # CLI entry point
├── internal/
│   ├── auth/               # Authentication providers (MSK, SASL)
│   ├── client/             # Kafka client management
│   ├── cmd/                # Command implementations
│   ├── config/             # Configuration management
│   ├── logger/             # Structured logging
│   ├── manager/            # Business logic (topics, groups, messages)
│   └── ui/                 # User interface (interactive mode, display)
├── pkg/types/              # Shared data types
├── Makefile               # Build automation
└── README.md              # This file
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for build automation)

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Development build (faster)
make dev-build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test ./internal/config -v
```

### Contributing

1. Fork the [repository](https://github.com/nipunap/kim)
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Run `make test` and `make lint`
6. Commit your changes (`git commit -m 'Add some amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a [Pull Request](https://github.com/nipunap/kim/pulls)

## Comparison with Kombucha

Kim is inspired by the Python Kombucha tool but offers several improvements:

| Feature | Kombucha (Python) | Kim (Go) |
|---------|------------------|----------|
| Performance | Slower startup | Fast startup and execution |
| Memory Usage | Higher | Lower memory footprint |
| Binary Size | Requires Python runtime | Single binary |
| Dependencies | Many Python packages | Minimal dependencies |
| Interactive Mode | Basic TUI | Enhanced with search and navigation |
| Output Formats | Limited | Table, JSON, YAML |
| Error Handling | Basic | Comprehensive with suggestions |
| Testing | Limited | Comprehensive test suite |
| Documentation | Basic | Extensive with examples |

## License

This project is licensed under the GPL-3.0 License - see the [LICENSE](https://github.com/nipunap/kim/blob/main/LICENSE) file for details.

## Acknowledgments

- Inspired by the original Kombucha Python tool
- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Uses [Sarama](https://github.com/IBM/sarama) for Kafka client functionality
- Interactive mode powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea)

## Support

- 📖 [Documentation](https://github.com/nipunap/kim/blob/main/README.md)
- 🐛 [Issue Tracker](https://github.com/nipunap/kim/issues)
- 💬 [Discussions](https://github.com/nipunap/kim/discussions)
- 📧 [Contact](https://github.com/nipunap)
