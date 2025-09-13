package testutil

import (
	"errors"
	"fmt"
	"time"

	"kim/internal/config"
	"kim/internal/logger"
	"kim/pkg/types"

	"github.com/IBM/sarama"
)

// MockClient implements a mock Kafka client for testing
type MockClient struct {
	connected      bool
	profile        *config.Profile
	logger         *logger.Logger
	topics         map[string]*sarama.TopicMetadata
	groups         map[string]*sarama.GroupDescription
	shouldFailPing bool
	shouldFailOps  bool
}

// NewMockClient creates a new mock client
func NewMockClient(profile *config.Profile, log *logger.Logger) *MockClient {
	return &MockClient{
		connected: false,
		profile:   profile,
		logger:    log,
		topics:    make(map[string]*sarama.TopicMetadata),
		groups:    make(map[string]*sarama.GroupDescription),
	}
}

// Connect simulates connecting to Kafka
func (m *MockClient) Connect() error {
	if m.shouldFailOps {
		return errors.New("mock connection failed")
	}
	m.connected = true
	return nil
}

// Disconnect simulates disconnecting from Kafka
func (m *MockClient) Disconnect() error {
	m.connected = false
	return nil
}

// IsConnected returns connection status
func (m *MockClient) IsConnected() bool {
	return m.connected
}

// Ping simulates pinging the Kafka cluster
func (m *MockClient) Ping() error {
	if !m.connected {
		return errors.New("not connected")
	}
	if m.shouldFailPing {
		return errors.New("mock ping failed")
	}
	return nil
}

// GetProfile returns the profile
func (m *MockClient) GetProfile() *config.Profile {
	return m.profile
}

// Mock admin client methods
func (m *MockClient) ListTopics() (map[string]sarama.TopicDetail, error) {
	if m.shouldFailOps {
		return nil, errors.New("mock list topics failed")
	}

	topics := make(map[string]sarama.TopicDetail)
	for name, meta := range m.topics {
		topics[name] = sarama.TopicDetail{
			NumPartitions:     int32(len(meta.Partitions)),
			ReplicationFactor: int16(len(meta.Partitions[0].Replicas)),
		}
	}
	return topics, nil
}

func (m *MockClient) DescribeTopics(topics []string) ([]*sarama.TopicMetadata, error) {
	if m.shouldFailOps {
		return nil, errors.New("mock describe topics failed")
	}

	var result []*sarama.TopicMetadata
	for _, topicName := range topics {
		if meta, exists := m.topics[topicName]; exists {
			result = append(result, meta)
		}
	}
	return result, nil
}

func (m *MockClient) ListConsumerGroups() (map[string]string, error) {
	if m.shouldFailOps {
		return nil, errors.New("mock list groups failed")
	}

	groups := make(map[string]string)
	for groupID := range m.groups {
		groups[groupID] = "consumer"
	}
	return groups, nil
}

func (m *MockClient) DescribeConsumerGroups(groups []string) ([]*sarama.GroupDescription, error) {
	if m.shouldFailOps {
		return nil, errors.New("mock describe groups failed")
	}

	var result []*sarama.GroupDescription
	for _, groupID := range groups {
		if desc, exists := m.groups[groupID]; exists {
			result = append(result, desc)
		}
	}
	return result, nil
}

// Helper methods to set up mock data
func (m *MockClient) AddMockTopic(name string, partitions int, replicationFactor int) {
	partitionMeta := make([]*sarama.PartitionMetadata, partitions)
	for i := 0; i < partitions; i++ {
		replicas := make([]int32, replicationFactor)
		for j := 0; j < replicationFactor; j++ {
			replicas[j] = int32(j)
		}
		partitionMeta[i] = &sarama.PartitionMetadata{
			ID:       int32(i),
			Leader:   0,
			Replicas: replicas,
			Isr:      replicas,
		}
	}

	m.topics[name] = &sarama.TopicMetadata{
		Name:       name,
		Partitions: partitionMeta,
	}
}

func (m *MockClient) AddMockGroup(groupID, state, protocolType string, memberCount int) {
	members := make(map[string]*sarama.GroupMemberDescription)
	for i := 0; i < memberCount; i++ {
		memberID := fmt.Sprintf("member-%d", i)
		members[memberID] = &sarama.GroupMemberDescription{
			MemberId:   memberID,
			ClientId:   fmt.Sprintf("client-%d", i),
			ClientHost: fmt.Sprintf("host-%d", i),
		}
	}

	m.groups[groupID] = &sarama.GroupDescription{
		GroupId:      groupID,
		State:        state,
		ProtocolType: protocolType,
		Protocol:     "range",
		Members:      members,
	}
}

func (m *MockClient) SetShouldFailPing(fail bool) {
	m.shouldFailPing = fail
}

func (m *MockClient) SetShouldFailOps(fail bool) {
	m.shouldFailOps = fail
}

// MockConsumerSession represents a mock consumer session
type MockConsumerSession struct {
	Topic     string
	Partition int32
	GroupID   string
	Messages  chan *types.Message
	Errors    chan error
	Active    bool
}

// NewMockConsumerSession creates a new mock consumer session
func NewMockConsumerSession(topic string, partition int32, groupID string) *MockConsumerSession {
	return &MockConsumerSession{
		Topic:     topic,
		Partition: partition,
		GroupID:   groupID,
		Messages:  make(chan *types.Message, 100),
		Errors:    make(chan error, 10),
		Active:    true,
	}
}

// SendMockMessage sends a mock message to the consumer
func (s *MockConsumerSession) SendMockMessage(key, value string, headers map[string]string) {
	if !s.Active {
		return
	}

	msg := &types.Message{
		Topic:     s.Topic,
		Partition: s.Partition,
		Offset:    time.Now().UnixNano(), // Use timestamp as mock offset
		Key:       key,
		Value:     value,
		Headers:   headers,
		Timestamp: time.Now(),
	}

	select {
	case s.Messages <- msg:
	default:
		// Channel full, drop message
	}
}

// SendMockError sends a mock error to the consumer
func (s *MockConsumerSession) SendMockError(err error) {
	if !s.Active {
		return
	}

	select {
	case s.Errors <- err:
	default:
		// Channel full, drop error
	}
}

// Stop stops the mock consumer session
func (s *MockConsumerSession) Stop() {
	s.Active = false
	close(s.Messages)
	close(s.Errors)
}

// TestProfile creates a test Kafka profile
func TestProfile() *config.Profile {
	return &config.Profile{
		Name:             "test-kafka",
		Type:             "kafka",
		BootstrapServers: "localhost:9092",
		SecurityProtocol: "PLAINTEXT",
	}
}

// TestMSKProfile creates a test MSK profile
func TestMSKProfile() *config.Profile {
	return &config.Profile{
		Name:       "test-msk",
		Type:       "msk",
		Region:     "us-east-1",
		ClusterARN: "arn:aws:kafka:us-east-1:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1",
		AuthMethod: "IAM",
	}
}

// TestConfig creates a test configuration
func TestConfig() *config.Config {
	return &config.Config{
		ActiveProfile: "test-kafka",
		Profiles: map[string]*config.Profile{
			"test-kafka": TestProfile(),
			"test-msk":   TestMSKProfile(),
		},
		Settings: &config.Settings{
			PageSize:        20,
			RefreshInterval: 10,
			DefaultFormat:   "table",
			ColorScheme:     "default",
			VimMode:         false,
		},
	}
}

// TestLogger creates a test logger
func TestLogger() *logger.Logger {
	return logger.New()
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t interface{ Fatalf(string, ...interface{}) }, err error) {
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t interface{ Fatalf(string, ...interface{}) }, err error) {
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

// AssertEqual fails the test if expected != actual
func AssertEqual(t interface{ Errorf(string, ...interface{}) }, expected, actual interface{}) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEqual fails the test if expected == actual
func AssertNotEqual(t interface{ Errorf(string, ...interface{}) }, expected, actual interface{}) {
	if expected == actual {
		t.Errorf("Expected %v to not equal %v", expected, actual)
	}
}

// AssertTrue fails the test if condition is false
func AssertTrue(t interface{ Errorf(string, ...interface{}) }, condition bool, message string) {
	if !condition {
		t.Errorf("Expected true: %s", message)
	}
}

// AssertFalse fails the test if condition is true
func AssertFalse(t interface{ Errorf(string, ...interface{}) }, condition bool, message string) {
	if condition {
		t.Errorf("Expected false: %s", message)
	}
}
