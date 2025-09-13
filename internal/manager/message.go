package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nipunap/kim/internal/client"
	"github.com/nipunap/kim/internal/logger"
	"github.com/nipunap/kim/pkg/types"

	"github.com/IBM/sarama"
)

// MessageManager manages Kafka message operations
type MessageManager struct {
	client    *client.Client
	logger    *logger.Logger
	consumers map[string]*ConsumerSession
	mutex     sync.RWMutex
}

// ConsumerSession represents an active consumer session
type ConsumerSession struct {
	Consumer      sarama.PartitionConsumer
	Topic         string
	Partition     int32
	GroupID       string
	Messages      chan *types.Message
	Errors        chan error
	Stop          chan struct{}
	FromBeginning bool
}

// NewMessageManager creates a new message manager
func NewMessageManager(client *client.Client, logger *logger.Logger) *MessageManager {
	return &MessageManager{
		client:    client,
		logger:    logger,
		consumers: make(map[string]*ConsumerSession),
	}
}

// ProduceMessage produces a message to a topic
func (mm *MessageManager) ProduceMessage(ctx context.Context, req *types.ProduceRequest) (*types.ProduceResponse, error) {
	if !mm.client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Create the message
	msg := &sarama.ProducerMessage{
		Topic: req.Topic,
		Value: sarama.StringEncoder(req.Value),
	}

	// Add key if provided
	if req.Key != "" {
		msg.Key = sarama.StringEncoder(req.Key)
	}

	// Add partition if specified
	if req.Partition != nil {
		msg.Partition = *req.Partition
	}

	// Add headers if provided
	if len(req.Headers) > 0 {
		msg.Headers = make([]sarama.RecordHeader, 0, len(req.Headers))
		for key, value := range req.Headers {
			msg.Headers = append(msg.Headers, sarama.RecordHeader{
				Key:   []byte(key),
				Value: []byte(value),
			})
		}
	}

	// Send the message
	partition, offset, err := mm.client.Producer.SendMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to produce message: %w", err)
	}

	mm.logger.Info("Message produced successfully",
		"topic", req.Topic, "partition", partition, "offset", offset)

	return &types.ProduceResponse{
		Topic:     req.Topic,
		Partition: partition,
		Offset:    offset,
		Timestamp: time.Now(),
	}, nil
}

// StartConsumer starts consuming messages from a topic
func (mm *MessageManager) StartConsumer(ctx context.Context, req *types.ConsumeRequest) (<-chan *types.Message, <-chan error, error) {
	if !mm.client.IsConnected() {
		return nil, nil, fmt.Errorf("client not connected")
	}

	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	sessionKey := fmt.Sprintf("%s-%s-%d", req.Topic, req.GroupID, req.Partition)

	// Check if consumer already exists
	if session, exists := mm.consumers[sessionKey]; exists {
		return session.Messages, session.Errors, nil
	}

	// Determine starting offset
	var offset int64
	if req.FromBeginning {
		offset = sarama.OffsetOldest
	} else {
		offset = sarama.OffsetNewest
	}

	// Create partition consumer
	partitionConsumer, err := mm.client.Consumer.ConsumePartition(req.Topic, req.Partition, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create partition consumer: %w", err)
	}

	// Create consumer session
	session := &ConsumerSession{
		Consumer:      partitionConsumer,
		Topic:         req.Topic,
		Partition:     req.Partition,
		GroupID:       req.GroupID,
		Messages:      make(chan *types.Message, 100),
		Errors:        make(chan error, 10),
		Stop:          make(chan struct{}),
		FromBeginning: req.FromBeginning,
	}

	mm.consumers[sessionKey] = session

	// Start consuming in a goroutine
	go mm.consumeMessages(session)

	mm.logger.Info("Started consumer",
		"topic", req.Topic, "partition", req.Partition, "group", req.GroupID)

	return session.Messages, session.Errors, nil
}

// consumeMessages handles the message consumption loop
func (mm *MessageManager) consumeMessages(session *ConsumerSession) {
	defer func() {
		close(session.Messages)
		close(session.Errors)
		session.Consumer.Close()

		mm.mutex.Lock()
		sessionKey := fmt.Sprintf("%s-%s-%d", session.Topic, session.GroupID, session.Partition)
		delete(mm.consumers, sessionKey)
		mm.mutex.Unlock()
	}()

	for {
		select {
		case msg := <-session.Consumer.Messages():
			if msg == nil {
				return
			}

			// Convert to our message type
			message := &types.Message{
				Topic:     msg.Topic,
				Partition: msg.Partition,
				Offset:    msg.Offset,
				Timestamp: msg.Timestamp,
				Key:       string(msg.Key),
				Value:     mm.formatMessageValue(msg.Value),
				Headers:   make(map[string]string),
			}

			// Convert headers
			for _, header := range msg.Headers {
				message.Headers[string(header.Key)] = string(header.Value)
			}

			select {
			case session.Messages <- message:
			case <-session.Stop:
				return
			}

		case err := <-session.Consumer.Errors():
			if err == nil {
				return
			}

			select {
			case session.Errors <- err:
			case <-session.Stop:
				return
			}

		case <-session.Stop:
			return
		}
	}
}

// formatMessageValue attempts to format the message value for display
func (mm *MessageManager) formatMessageValue(value []byte) string {
	if len(value) == 0 {
		return ""
	}

	// Try to parse as JSON first
	var jsonObj interface{}
	if err := json.Unmarshal(value, &jsonObj); err == nil {
		// Pretty print JSON
		if formatted, err := json.MarshalIndent(jsonObj, "", "  "); err == nil {
			return string(formatted)
		}
	}

	// Return as string if not JSON
	return string(value)
}

// StopConsumer stops a specific consumer
func (mm *MessageManager) StopConsumer(topic, groupID string, partition int32) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	sessionKey := fmt.Sprintf("%s-%s-%d", topic, groupID, partition)

	session, exists := mm.consumers[sessionKey]
	if !exists {
		return fmt.Errorf("consumer not found")
	}

	close(session.Stop)
	delete(mm.consumers, sessionKey)

	mm.logger.Info("Stopped consumer",
		"topic", topic, "partition", partition, "group", groupID)

	return nil
}

// StopAllConsumers stops all active consumers
func (mm *MessageManager) StopAllConsumers() error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	for sessionKey, session := range mm.consumers {
		close(session.Stop)
		delete(mm.consumers, sessionKey)
	}

	mm.logger.Info("Stopped all consumers")
	return nil
}

// GetActiveConsumers returns information about active consumers
func (mm *MessageManager) GetActiveConsumers() []*types.ConsumerInfo {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	consumers := make([]*types.ConsumerInfo, 0, len(mm.consumers))
	for _, session := range mm.consumers {
		consumers = append(consumers, &types.ConsumerInfo{
			Topic:         session.Topic,
			Partition:     session.Partition,
			GroupID:       session.GroupID,
			FromBeginning: session.FromBeginning,
		})
	}

	return consumers
}

// GetTopicMessages retrieves messages from a topic with pagination
func (mm *MessageManager) GetTopicMessages(ctx context.Context, req *types.GetMessagesRequest) (*types.MessageList, error) {
	if !mm.client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// This is a simplified implementation that would need to be enhanced
	// for production use with proper offset management and pagination

	var messages []*types.Message
	var offset int64

	if req.FromBeginning {
		offset = sarama.OffsetOldest
	} else {
		offset = sarama.OffsetNewest
	}

	// Create a temporary consumer for fetching messages
	partitionConsumer, err := mm.client.Consumer.ConsumePartition(req.Topic, req.Partition, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to create partition consumer: %w", err)
	}
	defer partitionConsumer.Close()

	// Collect messages with timeout
	timeout := time.After(5 * time.Second)
	messageCount := 0
	maxMessages := req.Limit
	if maxMessages == 0 {
		maxMessages = 100 // Default limit
	}

	for messageCount < maxMessages {
		select {
		case msg := <-partitionConsumer.Messages():
			if msg == nil {
				break
			}

			message := &types.Message{
				Topic:     msg.Topic,
				Partition: msg.Partition,
				Offset:    msg.Offset,
				Timestamp: msg.Timestamp,
				Key:       string(msg.Key),
				Value:     mm.formatMessageValue(msg.Value),
				Headers:   make(map[string]string),
			}

			// Convert headers
			for _, header := range msg.Headers {
				message.Headers[string(header.Key)] = string(header.Value)
			}

			messages = append(messages, message)
			messageCount++

		case err := <-partitionConsumer.Errors():
			if err != nil {
				return nil, fmt.Errorf("consumer error: %w", err)
			}

		case <-timeout:
			break

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &types.MessageList{
		Messages: messages,
		Pagination: &types.Pagination{
			CurrentPage: 1,
			TotalPages:  1,
			PageSize:    len(messages),
			TotalItems:  len(messages),
		},
	}, nil
}
