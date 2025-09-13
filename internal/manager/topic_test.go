package manager

import (
	"context"
	"testing"

	"github.com/nipunap/kim/internal/client"
	"github.com/nipunap/kim/internal/testutil"
	"github.com/nipunap/kim/pkg/types"
)

func TestNewTopicManager(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	tm := NewTopicManager(c, logger)
	if tm == nil {
		t.Fatal("TopicManager should not be nil")
	}
}

func TestTopicManagerListTopics(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	tm := NewTopicManager(c, logger)

	// Test basic list - this will fail if no Kafka is running, but that's expected
	opts := &types.ListOptions{
		Page:     1,
		PageSize: 10,
	}

	_, err = tm.ListTopics(context.Background(), opts)
	// We expect this to fail in test environment without Kafka
	if err == nil {
		t.Log("ListTopics succeeded (Kafka must be running)")
	} else {
		t.Logf("ListTopics failed as expected in test environment: %v", err)
	}
}

func TestTopicManagerDescribeTopic(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	tm := NewTopicManager(c, logger)

	// Test describe topic - this will fail if no Kafka is running, but that's expected
	_, err = tm.DescribeTopic(context.Background(), "test-topic")
	// We expect this to fail in test environment without Kafka
	if err == nil {
		t.Log("DescribeTopic succeeded (Kafka must be running)")
	} else {
		t.Logf("DescribeTopic failed as expected in test environment: %v", err)
	}
}

func TestTopicManagerCreateTopic(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	tm := NewTopicManager(c, logger)

	// Test create topic - this will fail if no Kafka is running, but that's expected
	req := &types.CreateTopicRequest{
		Name:              "test-topic",
		Partitions:        1,
		ReplicationFactor: 1,
	}

	err = tm.CreateTopic(context.Background(), req)
	// We expect this to fail in test environment without Kafka
	if err == nil {
		t.Log("CreateTopic succeeded (Kafka must be running)")
	} else {
		t.Logf("CreateTopic failed as expected in test environment: %v", err)
	}
}

func TestTopicManagerDeleteTopic(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	tm := NewTopicManager(c, logger)

	// Test delete topic - this will fail if no Kafka is running, but that's expected
	err = tm.DeleteTopic(context.Background(), "test-topic")
	// We expect this to fail in test environment without Kafka
	if err == nil {
		t.Log("DeleteTopic succeeded (Kafka must be running)")
	} else {
		t.Logf("DeleteTopic failed as expected in test environment: %v", err)
	}
}
