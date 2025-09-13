package manager

import (
	"testing"

	"github.com/nipunap/kim/internal/testutil"
	"github.com/nipunap/kim/pkg/types"

	"github.com/IBM/sarama"
)

// MockTopicClient implements the client interface for topic testing
type MockTopicClient struct {
	*testutil.MockClient
	adminClient *MockAdminClient
}

type MockAdminClient struct {
	topics     map[string]sarama.TopicDetail
	configs    map[string]map[string]string
	shouldFail bool
}

func (m *MockAdminClient) ListTopics() (map[string]sarama.TopicDetail, error) {
	return m.topics, nil
}

func (m *MockAdminClient) DescribeTopics(topics []string) ([]*sarama.TopicMetadata, error) {
	var result []*sarama.TopicMetadata
	for _, topicName := range topics {
		if detail, exists := m.topics[topicName]; exists {
			partitions := make([]*sarama.PartitionMetadata, detail.NumPartitions)
			for i := int32(0); i < detail.NumPartitions; i++ {
				replicas := make([]int32, detail.ReplicationFactor)
				for j := int16(0); j < detail.ReplicationFactor; j++ {
					replicas[j] = int32(j)
				}
				partitions[i] = &sarama.PartitionMetadata{
					ID:       i,
					Leader:   0,
					Replicas: replicas,
					Isr:      replicas,
				}
			}
			result = append(result, &sarama.TopicMetadata{
				Name:       topicName,
				Partitions: partitions,
			})
		}
	}
	return result, nil
}

func (m *MockAdminClient) DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error) {
	if configs, exists := m.configs[resource.Name]; exists {
		var entries []sarama.ConfigEntry
		for key, value := range configs {
			entries = append(entries, sarama.ConfigEntry{
				Name:  key,
				Value: value,
			})
		}
		return entries, nil
	}
	return []sarama.ConfigEntry{}, nil
}

func (m *MockAdminClient) CreateTopic(topic string, detail *sarama.TopicDetail, validateOnly bool) error {
	if m.shouldFail {
		return sarama.ErrTopicAlreadyExists
	}
	m.topics[topic] = *detail
	return nil
}

func (m *MockAdminClient) DeleteTopic(topic string) error {
	if m.shouldFail {
		return sarama.ErrUnknownTopicOrPartition
	}
	delete(m.topics, topic)
	return nil
}

func newMockTopicClient() *MockTopicClient {
	mockClient := testutil.NewMockClient(testutil.TestProfile(), testutil.TestLogger())
	adminClient := &MockAdminClient{
		topics: map[string]sarama.TopicDetail{
			"test-topic-1": {
				NumPartitions:     3,
				ReplicationFactor: 2,
			},
			"test-topic-2": {
				NumPartitions:     1,
				ReplicationFactor: 1,
			},
			"internal-topic": {
				NumPartitions:     1,
				ReplicationFactor: 1,
			},
		},
		configs: map[string]map[string]string{
			"test-topic-1": {
				"retention.ms":   "604800000",
				"cleanup.policy": "delete",
			},
		},
	}

	// Add topics to mock client as well
	mockClient.AddMockTopic("test-topic-1", 3, 2)
	mockClient.AddMockTopic("test-topic-2", 1, 1)
	mockClient.AddMockTopic("internal-topic", 1, 1)

	return &MockTopicClient{
		MockClient:  mockClient,
		adminClient: adminClient,
	}
}

func TestTopicManagerListTopics(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	// Test basic list
	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 10,
		Format:   "table",
	}

	result, err := tm.ListTopics(opts)
	testutil.AssertNoError(t, err)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if len(result.Topics) == 0 {
		t.Error("Should return topics")
	}

	// Verify pagination
	if result.Pagination == nil {
		t.Error("Pagination should not be nil")
	}

	if result.Pagination.TotalItems != 3 {
		t.Errorf("Expected 3 total items, got %d", result.Pagination.TotalItems)
	}
}

func TestTopicManagerListTopicsWithFilter(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	// Test with filter
	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 10,
		Format:   "table",
		Filter:   "test-topic-1",
	}

	result, err := tm.ListTopics(opts)
	testutil.AssertNoError(t, err)

	if len(result.Topics) != 1 {
		t.Errorf("Expected 1 filtered topic, got %d", len(result.Topics))
	}

	if result.Topics[0].Name != "test-topic-1" {
		t.Errorf("Expected topic name 'test-topic-1', got '%s'", result.Topics[0].Name)
	}
}

func TestTopicManagerListTopicsWithPagination(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	// Test pagination - page 1 with size 2
	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 2,
		Format:   "table",
	}

	result, err := tm.ListTopics(opts)
	testutil.AssertNoError(t, err)

	if len(result.Topics) != 2 {
		t.Errorf("Expected 2 topics on page 1, got %d", len(result.Topics))
	}

	if result.Pagination.CurrentPage != 1 {
		t.Errorf("Expected current page 1, got %d", result.Pagination.CurrentPage)
	}

	if result.Pagination.TotalPages != 2 {
		t.Errorf("Expected 2 total pages, got %d", result.Pagination.TotalPages)
	}

	// Test page 2
	opts.Page = 2
	result, err = tm.ListTopics(opts)
	testutil.AssertNoError(t, err)

	if len(result.Topics) != 1 {
		t.Errorf("Expected 1 topic on page 2, got %d", len(result.Topics))
	}
}

func TestTopicManagerDescribeTopic(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	// Test describe existing topic
	result, err := tm.DescribeTopic("test-topic-1")
	testutil.AssertNoError(t, err)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Name != "test-topic-1" {
		t.Errorf("Expected topic name 'test-topic-1', got '%s'", result.Name)
	}

	if result.Partitions != 3 {
		t.Errorf("Expected 3 partitions, got %d", result.Partitions)
	}

	if result.ReplicationFactor != 2 {
		t.Errorf("Expected replication factor 2, got %d", result.ReplicationFactor)
	}

	if len(result.PartitionDetails) != 3 {
		t.Errorf("Expected 3 partition details, got %d", len(result.PartitionDetails))
	}

	// Verify partition details
	for i, partition := range result.PartitionDetails {
		if partition.ID != int32(i) {
			t.Errorf("Expected partition ID %d, got %d", i, partition.ID)
		}
		if len(partition.Replicas) != 2 {
			t.Errorf("Expected 2 replicas for partition %d, got %d", i, len(partition.Replicas))
		}
	}

	// Test describe non-existent topic
	_, err = tm.DescribeTopic("non-existent")
	testutil.AssertError(t, err)
}

func TestTopicManagerCreateTopic(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	// Test create topic
	req := &types.CreateTopicRequest{
		Name:              "new-topic",
		Partitions:        2,
		ReplicationFactor: 1,
		Configs: map[string]string{
			"retention.ms": "86400000",
		},
	}

	err := tm.CreateTopic(req)
	testutil.AssertNoError(t, err)

	// Verify topic was created
	result, err := tm.DescribeTopic("new-topic")
	testutil.AssertNoError(t, err)

	if result.Name != "new-topic" {
		t.Errorf("Expected topic name 'new-topic', got '%s'", result.Name)
	}

	if result.Partitions != 2 {
		t.Errorf("Expected 2 partitions, got %d", result.Partitions)
	}
}

func TestTopicManagerCreateTopicFailure(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()
	mockClient.adminClient.shouldFail = true

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	req := &types.CreateTopicRequest{
		Name:              "existing-topic",
		Partitions:        1,
		ReplicationFactor: 1,
	}

	err := tm.CreateTopic(req)
	testutil.AssertError(t, err)
}

func TestTopicManagerDeleteTopic(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	// Test delete existing topic
	err := tm.DeleteTopic("test-topic-2")
	testutil.AssertNoError(t, err)

	// Verify topic was deleted
	_, err = tm.DescribeTopic("test-topic-2")
	testutil.AssertError(t, err)
}

func TestTopicManagerDeleteTopicFailure(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()
	mockClient.adminClient.shouldFail = true

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	err := tm.DeleteTopic("non-existent")
	testutil.AssertError(t, err)
}

func TestTopicManagerGetTopicOffsets(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	// Test get offsets
	offsets, err := tm.GetTopicOffsets("test-topic-1")
	testutil.AssertNoError(t, err)

	if len(offsets) != 3 {
		t.Errorf("Expected offsets for 3 partitions, got %d", len(offsets))
	}

	// Verify all partitions have offsets
	for i := int32(0); i < 3; i++ {
		if _, exists := offsets[i]; !exists {
			t.Errorf("Missing offset for partition %d", i)
		}
	}
}

func TestTopicManagerSorting(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	// Test sort by name ascending
	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 10,
		Format:   "table",
		SortBy:   "name",
		SortDesc: false,
	}

	result, err := tm.ListTopics(opts)
	testutil.AssertNoError(t, err)

	// Verify topics are sorted by name
	for i := 1; i < len(result.Topics); i++ {
		if result.Topics[i-1].Name > result.Topics[i].Name {
			t.Error("Topics should be sorted by name ascending")
		}
	}

	// Test sort by partitions descending
	opts.SortBy = "partitions"
	opts.SortDesc = true

	result, err = tm.ListTopics(opts)
	testutil.AssertNoError(t, err)

	// Verify topics are sorted by partitions descending
	for i := 1; i < len(result.Topics); i++ {
		if result.Topics[i-1].Partitions < result.Topics[i].Partitions {
			t.Error("Topics should be sorted by partitions descending")
		}
	}
}

func TestTopicManagerNotConnected(t *testing.T) {
	mockClient := newMockTopicClient()
	// Don't connect

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 10,
		Format:   "table",
	}

	_, err := tm.ListTopics(opts)
	testutil.AssertError(t, err)
}

func TestTopicManagerInvalidRequests(t *testing.T) {
	mockClient := newMockTopicClient()
	mockClient.Connect()

	tm := NewTopicManager(mockClient, testutil.TestLogger())

	// Test create topic with invalid request
	invalidReq := &types.CreateTopicRequest{
		Name:       "", // Empty name
		Partitions: 0,  // Invalid partitions
	}

	err := tm.CreateTopic(invalidReq)
	testutil.AssertError(t, err)

	// Test delete topic with empty name
	err = tm.DeleteTopic("")
	testutil.AssertError(t, err)

	// Test describe topic with empty name
	_, err = tm.DescribeTopic("")
	testutil.AssertError(t, err)
}
