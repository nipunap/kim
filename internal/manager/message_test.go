package manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/nipunap/kim/internal/testutil"
	"github.com/nipunap/kim/pkg/types"

	"github.com/IBM/sarama"
)

// MockMessageClient implements the client interface for message testing
type MockMessageClient struct {
	*testutil.MockClient
	producer *MockSyncProducer
	consumer *MockConsumer
}

type MockSyncProducer struct {
	messages   []sarama.ProducerMessage
	shouldFail bool
}

func (m *MockSyncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	if m.shouldFail {
		return 0, 0, sarama.ErrNotEnoughReplicas
	}

	m.messages = append(m.messages, *msg)
	return 0, int64(len(m.messages)), nil
}

func (m *MockSyncProducer) Close() error {
	return nil
}

type MockConsumer struct {
	partitionConsumers map[string]*MockPartitionConsumer
	shouldFail         bool
}

func (m *MockConsumer) ConsumePartition(topic string, partition int32, offset int64) (sarama.PartitionConsumer, error) {
	if m.shouldFail {
		return nil, sarama.ErrOffsetOutOfRange
	}

	key := fmt.Sprintf("%s-%d", topic, partition)
	if pc, exists := m.partitionConsumers[key]; exists {
		return pc, nil
	}

	pc := &MockPartitionConsumer{
		messages: make(chan *sarama.ConsumerMessage, 100),
		errors:   make(chan *sarama.ConsumerError, 10),
		closed:   make(chan struct{}),
	}
	m.partitionConsumers[key] = pc
	return pc, nil
}

func (m *MockConsumer) Close() error {
	for _, pc := range m.partitionConsumers {
		pc.Close()
	}
	return nil
}

type MockPartitionConsumer struct {
	messages chan *sarama.ConsumerMessage
	errors   chan *sarama.ConsumerError
	closed   chan struct{}
}

func (m *MockPartitionConsumer) Messages() <-chan *sarama.ConsumerMessage {
	return m.messages
}

func (m *MockPartitionConsumer) Errors() <-chan *sarama.ConsumerError {
	return m.errors
}

func (m *MockPartitionConsumer) Close() error {
	close(m.closed)
	close(m.messages)
	close(m.errors)
	return nil
}

func (m *MockPartitionConsumer) SendMockMessage(topic string, partition int32, key, value string) {
	msg := &sarama.ConsumerMessage{
		Topic:     topic,
		Partition: partition,
		Offset:    time.Now().UnixNano(),
		Key:       []byte(key),
		Value:     []byte(value),
		Timestamp: time.Now(),
	}

	select {
	case m.messages <- msg:
	case <-m.closed:
	}
}

func newMockMessageClient() *MockMessageClient {
	mockClient := testutil.NewMockClient(testutil.TestProfile(), testutil.TestLogger())

	producer := &MockSyncProducer{
		messages: make([]sarama.ProducerMessage, 0),
	}

	consumer := &MockConsumer{
		partitionConsumers: make(map[string]*MockPartitionConsumer),
	}

	return &MockMessageClient{
		MockClient: mockClient,
		producer:   producer,
		consumer:   consumer,
	}
}

func TestMessageManagerProduce(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	// Test basic message production
	req := &types.ProduceRequest{
		Topic: "test-topic",
		Key:   "test-key",
		Value: "test-value",
		Headers: map[string]string{
			"header1": "value1",
			"header2": "value2",
		},
	}

	err := mm.Produce(req)
	testutil.AssertNoError(t, err)

	// Verify message was sent
	if len(mockClient.producer.messages) != 1 {
		t.Errorf("Expected 1 message sent, got %d", len(mockClient.producer.messages))
	}

	sentMsg := mockClient.producer.messages[0]
	if sentMsg.Topic != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got '%s'", sentMsg.Topic)
	}

	if string(sentMsg.Key) != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", string(sentMsg.Key))
	}

	if string(sentMsg.Value) != "test-value" {
		t.Errorf("Expected value 'test-value', got '%s'", string(sentMsg.Value))
	}

	// Verify headers
	if len(sentMsg.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(sentMsg.Headers))
	}
}

func TestMessageManagerProduceWithoutKey(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	// Test message production without key
	req := &types.ProduceRequest{
		Topic: "test-topic",
		Value: "test-value-no-key",
	}

	err := mm.Produce(req)
	testutil.AssertNoError(t, err)

	sentMsg := mockClient.producer.messages[0]
	if sentMsg.Key != nil {
		t.Error("Expected nil key when not provided")
	}
}

func TestMessageManagerProduceFailure(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()
	mockClient.producer.shouldFail = true

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	req := &types.ProduceRequest{
		Topic: "test-topic",
		Value: "test-value",
	}

	err := mm.Produce(req)
	testutil.AssertError(t, err)
}

func TestMessageManagerConsume(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	// Test start consuming
	req := &types.ConsumeRequest{
		Topic:     "test-topic",
		Partition: 0,
		GroupID:   "test-group",
		Offset:    sarama.OffsetNewest,
	}

	session, err := mm.StartConsumer(req)
	testutil.AssertNoError(t, err)

	if session == nil {
		t.Fatal("Session should not be nil")
	}

	if session.Topic != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got '%s'", session.Topic)
	}

	if session.Partition != 0 {
		t.Errorf("Expected partition 0, got %d", session.Partition)
	}

	if session.GroupID != "test-group" {
		t.Errorf("Expected group ID 'test-group', got '%s'", session.GroupID)
	}

	// Test that channels are created
	if session.Messages == nil {
		t.Error("Messages channel should not be nil")
	}

	if session.Errors == nil {
		t.Error("Errors channel should not be nil")
	}

	// Clean up
	mm.StopConsumer("test-topic", "test-group", 0)
}

func TestMessageManagerConsumeMessages(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	req := &types.ConsumeRequest{
		Topic:     "test-topic",
		Partition: 0,
		GroupID:   "test-group",
		Offset:    sarama.OffsetOldest,
	}

	session, err := mm.StartConsumer(req)
	testutil.AssertNoError(t, err)

	// Get the partition consumer and send a mock message
	key := fmt.Sprintf("%s-%d", "test-topic", 0)
	pc := mockClient.consumer.partitionConsumers[key]

	// Send a mock message
	go func() {
		time.Sleep(100 * time.Millisecond) // Give consumer time to start
		pc.SendMockMessage("test-topic", 0, "test-key", "test-value")
	}()

	// Wait for message
	select {
	case msg := <-session.Messages:
		if msg == nil {
			t.Fatal("Received nil message")
		}
		if msg.Topic != "test-topic" {
			t.Errorf("Expected topic 'test-topic', got '%s'", msg.Topic)
		}
		if msg.Key != "test-key" {
			t.Errorf("Expected key 'test-key', got '%s'", msg.Key)
		}
		if msg.Value != "test-value" {
			t.Errorf("Expected value 'test-value', got '%s'", msg.Value)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message")
	}

	// Clean up
	mm.StopConsumer("test-topic", "test-group", 0)
}

func TestMessageManagerStopConsumer(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	req := &types.ConsumeRequest{
		Topic:     "test-topic",
		Partition: 0,
		GroupID:   "test-group",
		Offset:    sarama.OffsetNewest,
	}

	session, err := mm.StartConsumer(req)
	testutil.AssertNoError(t, err)

	// Stop consumer
	err = mm.StopConsumer("test-topic", "test-group", 0)
	testutil.AssertNoError(t, err)

	// Verify session is stopped
	select {
	case _, ok := <-session.Messages:
		if ok {
			t.Error("Messages channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
		// Timeout is acceptable as channel might be closed already
	}
}

func TestMessageManagerStopNonExistentConsumer(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	// Try to stop non-existent consumer
	err := mm.StopConsumer("non-existent", "group", 0)
	testutil.AssertError(t, err)
}

func TestMessageManagerConsumeFailure(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()
	mockClient.consumer.shouldFail = true

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	req := &types.ConsumeRequest{
		Topic:     "test-topic",
		Partition: 0,
		GroupID:   "test-group",
		Offset:    sarama.OffsetNewest,
	}

	_, err := mm.StartConsumer(req)
	testutil.AssertError(t, err)
}

func TestMessageManagerGetMessages(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	// Test get messages (simplified implementation)
	req := &types.ConsumeRequest{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    sarama.OffsetOldest,
		Limit:     10,
	}

	messages, err := mm.GetMessages(req)
	testutil.AssertNoError(t, err)

	// In our mock implementation, this returns empty messages
	if messages == nil {
		t.Error("Messages should not be nil")
	}

	// The mock implementation returns empty slice
	if len(messages) != 0 {
		t.Logf("Got %d messages (mock implementation)", len(messages))
	}
}

func TestMessageManagerNotConnected(t *testing.T) {
	mockClient := newMockMessageClient()
	// Don't connect

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	// Test produce when not connected
	req := &types.ProduceRequest{
		Topic: "test-topic",
		Value: "test-value",
	}

	err := mm.Produce(req)
	testutil.AssertError(t, err)

	// Test consume when not connected
	consumeReq := &types.ConsumeRequest{
		Topic:     "test-topic",
		Partition: 0,
		GroupID:   "test-group",
		Offset:    sarama.OffsetNewest,
	}

	_, err = mm.StartConsumer(consumeReq)
	testutil.AssertError(t, err)
}

func TestMessageManagerInvalidRequests(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	// Test produce with empty topic
	req := &types.ProduceRequest{
		Topic: "",
		Value: "test-value",
	}

	err := mm.Produce(req)
	testutil.AssertError(t, err)

	// Test produce with empty value
	req = &types.ProduceRequest{
		Topic: "test-topic",
		Value: "",
	}

	err = mm.Produce(req)
	testutil.AssertError(t, err)

	// Test consume with empty topic
	consumeReq := &types.ConsumeRequest{
		Topic:     "",
		Partition: 0,
		GroupID:   "test-group",
		Offset:    sarama.OffsetNewest,
	}

	_, err = mm.StartConsumer(consumeReq)
	testutil.AssertError(t, err)

	// Test consume with empty group ID
	consumeReq = &types.ConsumeRequest{
		Topic:     "test-topic",
		Partition: 0,
		GroupID:   "",
		Offset:    sarama.OffsetNewest,
	}

	_, err = mm.StartConsumer(consumeReq)
	testutil.AssertError(t, err)
}

func TestMessageManagerMultipleConsumers(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	// Start multiple consumers
	req1 := &types.ConsumeRequest{
		Topic:     "topic1",
		Partition: 0,
		GroupID:   "group1",
		Offset:    sarama.OffsetNewest,
	}

	req2 := &types.ConsumeRequest{
		Topic:     "topic2",
		Partition: 1,
		GroupID:   "group2",
		Offset:    sarama.OffsetNewest,
	}

	session1, err := mm.StartConsumer(req1)
	testutil.AssertNoError(t, err)

	session2, err := mm.StartConsumer(req2)
	testutil.AssertNoError(t, err)

	// Verify both sessions are different
	if session1 == session2 {
		t.Error("Sessions should be different")
	}

	// Stop both consumers
	err = mm.StopConsumer("topic1", "group1", 0)
	testutil.AssertNoError(t, err)

	err = mm.StopConsumer("topic2", "group2", 1)
	testutil.AssertNoError(t, err)
}

func TestMessageManagerDuplicateConsumer(t *testing.T) {
	mockClient := newMockMessageClient()
	mockClient.Connect()

	mm := NewMessageManager(mockClient, testutil.TestLogger())

	req := &types.ConsumeRequest{
		Topic:     "test-topic",
		Partition: 0,
		GroupID:   "test-group",
		Offset:    sarama.OffsetNewest,
	}

	// Start first consumer
	session1, err := mm.StartConsumer(req)
	testutil.AssertNoError(t, err)

	// Try to start duplicate consumer (should return existing session)
	session2, err := mm.StartConsumer(req)
	testutil.AssertNoError(t, err)

	// Should return the same session
	if session1 != session2 {
		t.Error("Should return existing session for duplicate consumer")
	}

	// Clean up
	mm.StopConsumer("test-topic", "test-group", 0)
}
