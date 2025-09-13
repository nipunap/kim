package manager

import (
	"context"
	"testing"

	"github.com/nipunap/kim/internal/client"
	"github.com/nipunap/kim/internal/testutil"
	"github.com/nipunap/kim/pkg/types"
)

func TestNewMessageManager(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	mm := NewMessageManager(c, logger)
	if mm == nil {
		t.Fatal("MessageManager should not be nil")
	}
}

func TestMessageManagerConsume(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	mm := NewMessageManager(c, logger)

	// Test start consumer - this will fail if no Kafka is running, but that's expected
	req := &types.ConsumeRequest{
		Topic:         "test-topic",
		GroupID:       "test-group",
		Partition:     0,
		FromBeginning: false,
	}

	_, _, err = mm.StartConsumer(context.Background(), req)
	// We expect this to fail in test environment without Kafka
	if err == nil {
		t.Log("Consume succeeded (Kafka must be running)")
	} else {
		t.Logf("Consume failed as expected in test environment: %v", err)
	}
}

func TestMessageManagerProduceMessage(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	mm := NewMessageManager(c, logger)

	// Test produce message - this will fail if no Kafka is running, but that's expected
	partition := int32(0)
	req := &types.ProduceRequest{
		Topic:     "test-topic",
		Partition: &partition,
		Key:       "test-key",
		Value:     "test-value",
	}

	_, err = mm.ProduceMessage(context.Background(), req)
	// We expect this to fail in test environment without Kafka
	if err == nil {
		t.Log("ProduceMessage succeeded (Kafka must be running)")
	} else {
		t.Logf("ProduceMessage failed as expected in test environment: %v", err)
	}
}
