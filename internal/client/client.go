package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/nipunap/kim/internal/auth"
	"github.com/nipunap/kim/internal/config"
	"github.com/nipunap/kim/internal/logger"

	"github.com/IBM/sarama"
)

// Manager manages Kafka client connections
type Manager struct {
	logger  *logger.Logger
	clients map[string]*Client
	mutex   sync.RWMutex
}

// Client wraps Kafka client functionality
type Client struct {
	Config      *sarama.Config
	AdminClient sarama.ClusterAdmin
	Consumer    sarama.Consumer
	Producer    sarama.SyncProducer
	profile     *config.Profile
	logger      *logger.Logger
	connected   bool
	mutex       sync.RWMutex
}

// NewManager creates a new client manager
func NewManager(logger *logger.Logger) *Manager {
	return &Manager{
		logger:  logger,
		clients: make(map[string]*Client),
	}
}

// GetClient returns or creates a client for the given profile
func (m *Manager) GetClient(profile *config.Profile) (*Client, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	clientKey := fmt.Sprintf("%s_%s", profile.Type, profile.Name)

	if client, exists := m.clients[clientKey]; exists && client.connected {
		return client, nil
	}

	client, err := m.createClient(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	m.clients[clientKey] = client
	return client, nil
}

// createClient creates a new Kafka client based on the profile
func (m *Manager) createClient(profile *config.Profile) (*Client, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_1_0 // Compatible with most Kafka versions
	config.ClientID = "kim-client"

	// Configure based on profile type
	switch profile.Type {
	case "msk":
		if err := m.configureMSK(config, profile); err != nil {
			return nil, fmt.Errorf("failed to configure MSK client: %w", err)
		}
	case "kafka":
		if err := m.configureKafka(config, profile); err != nil {
			return nil, fmt.Errorf("failed to configure Kafka client: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported profile type: %s", profile.Type)
	}

	client := &Client{
		Config:  config,
		profile: profile,
		logger:  m.logger,
	}

	if err := client.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return client, nil
}

// configureMSK configures the client for MSK
func (m *Manager) configureMSK(config *sarama.Config, profile *config.Profile) error {
	// Get bootstrap brokers from MSK
	brokers, err := auth.GetMSKBootstrapBrokers(profile.Region, profile.ClusterARN)
	if err != nil {
		return fmt.Errorf("failed to get MSK bootstrap brokers: %w", err)
	}

	// Configure authentication
	authMethod := profile.AuthMethod
	if authMethod == "" {
		authMethod = "IAM" // Default to IAM
	}

	switch authMethod {
	case "IAM":
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		config.Net.SASL.TokenProvider = auth.NewMSKTokenProvider(profile.Region)
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: false,
		}
	case "SASL_SCRAM":
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		config.Net.SASL.User = profile.SASLUsername
		config.Net.SASL.Password = profile.SASLPassword
		config.Net.TLS.Enable = true
	default:
		return fmt.Errorf("unsupported MSK auth method: %s", authMethod)
	}

	// Store brokers in profile for connection
	profile.BootstrapServers = brokers
	return nil
}

// configureKafka configures the client for regular Kafka
func (m *Manager) configureKafka(config *sarama.Config, profile *config.Profile) error {
	// Configure security protocol
	switch profile.SecurityProtocol {
	case "PLAINTEXT", "":
		// No additional configuration needed
	case "SSL":
		config.Net.TLS.Enable = true
		if err := m.configureSSL(config, profile); err != nil {
			return fmt.Errorf("failed to configure SSL: %w", err)
		}
	case "SASL_PLAINTEXT":
		config.Net.SASL.Enable = true
		if err := m.configureSASL(config, profile); err != nil {
			return fmt.Errorf("failed to configure SASL: %w", err)
		}
	case "SASL_SSL":
		config.Net.TLS.Enable = true
		config.Net.SASL.Enable = true
		if err := m.configureSSL(config, profile); err != nil {
			return fmt.Errorf("failed to configure SSL: %w", err)
		}
		if err := m.configureSASL(config, profile); err != nil {
			return fmt.Errorf("failed to configure SASL: %w", err)
		}
	default:
		return fmt.Errorf("unsupported security protocol: %s", profile.SecurityProtocol)
	}

	return nil
}

// configureSSL configures SSL settings
func (m *Manager) configureSSL(config *sarama.Config, profile *config.Profile) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !profile.SSLCheckHostname,
	}

	if profile.SSLCAFile != "" {
		// Load CA certificate
		// Implementation would load the CA file
		m.logger.Debug("SSL CA file configured", "file", profile.SSLCAFile)
	}

	if profile.SSLCertFile != "" && profile.SSLKeyFile != "" {
		// Load client certificate and key
		// Implementation would load the cert and key files
		m.logger.Debug("SSL client certificate configured",
			"cert", profile.SSLCertFile, "key", profile.SSLKeyFile)
	}

	config.Net.TLS.Config = tlsConfig
	return nil
}

// configureSASL configures SASL settings
func (m *Manager) configureSASL(config *sarama.Config, profile *config.Profile) error {
	switch profile.SASLMechanism {
	case "PLAIN":
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	case "SCRAM-SHA-256":
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
	case "SCRAM-SHA-512":
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	case "GSSAPI":
		config.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
	default:
		return fmt.Errorf("unsupported SASL mechanism: %s", profile.SASLMechanism)
	}

	config.Net.SASL.User = profile.SASLUsername
	config.Net.SASL.Password = profile.SASLPassword
	return nil
}

// connect establishes connections to Kafka
func (c *Client) connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	brokers := []string{c.profile.BootstrapServers}

	// Create admin client
	adminClient, err := sarama.NewClusterAdmin(brokers, c.Config)
	if err != nil {
		return fmt.Errorf("failed to create admin client: %w", err)
	}
	c.AdminClient = adminClient

	// Create consumer
	consumer, err := sarama.NewConsumer(brokers, c.Config)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	c.Consumer = consumer

	// Create producer
	c.Config.Producer.Return.Successes = true
	c.Config.Producer.RequiredAcks = sarama.WaitForAll
	c.Config.Producer.Retry.Max = 3
	c.Config.Producer.Timeout = 10 * time.Second

	producer, err := sarama.NewSyncProducer(brokers, c.Config)
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}
	c.Producer = producer

	c.connected = true
	c.logger.Info("Successfully connected to Kafka cluster",
		"profile", c.profile.Name, "type", c.profile.Type)

	return nil
}

// Close closes all client connections
func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var errors []error

	if c.AdminClient != nil {
		if err := c.AdminClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close admin client: %w", err))
		}
	}

	if c.Consumer != nil {
		if err := c.Consumer.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close consumer: %w", err))
		}
	}

	if c.Producer != nil {
		if err := c.Producer.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close producer: %w", err))
		}
	}

	c.connected = false

	if len(errors) > 0 {
		return fmt.Errorf("errors closing client: %v", errors)
	}

	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// Ping tests the connection to the Kafka cluster
func (c *Client) Ping(ctx context.Context) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	// Try to get cluster metadata as a ping
	_, _, err := c.AdminClient.DescribeCluster()
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}
