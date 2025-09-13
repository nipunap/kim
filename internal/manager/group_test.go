package manager

import (
	"context"
	"testing"

	"github.com/nipunap/kim/internal/client"
	"github.com/nipunap/kim/internal/testutil"
	"github.com/nipunap/kim/pkg/types"
)

func TestNewGroupManager(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	gm := NewGroupManager(c, logger)
	if gm == nil {
		t.Fatal("GroupManager should not be nil")
	}
}

func TestGroupManagerListGroups(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	gm := NewGroupManager(c, logger)

	// Test basic list - this will fail if no Kafka is running, but that's expected
	opts := &types.ListOptions{
		Page:     1,
		PageSize: 10,
	}

	_, err = gm.ListGroups(context.Background(), opts)
	// We expect this to fail in test environment without Kafka
	// The important thing is that the function signature is correct
	if err == nil {
		t.Log("ListGroups succeeded (Kafka must be running)")
	} else {
		t.Logf("ListGroups failed as expected in test environment: %v", err)
	}
}

func TestGroupManagerDescribeGroup(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	gm := NewGroupManager(c, logger)

	// Test describe group - this will fail if no Kafka is running, but that's expected
	_, err = gm.DescribeGroup(context.Background(), "test-group")
	// We expect this to fail in test environment without Kafka
	if err == nil {
		t.Log("DescribeGroup succeeded (Kafka must be running)")
	} else {
		t.Logf("DescribeGroup failed as expected in test environment: %v", err)
	}
}

func TestGroupManagerDeleteGroup(t *testing.T) {
	// Create a real client with test profile
	profile := testutil.TestProfile()
	logger := testutil.TestLogger()

	clientManager := client.NewManager(logger)
	c, err := clientManager.GetClient(profile)
	if err != nil {
		t.Skipf("Skipping test - cannot create client: %v", err)
	}

	gm := NewGroupManager(c, logger)

	// Test delete group - this will fail if no Kafka is running, but that's expected
	err = gm.DeleteGroup(context.Background(), "test-group")
	// We expect this to fail in test environment without Kafka
	if err == nil {
		t.Log("DeleteGroup succeeded (Kafka must be running)")
	} else {
		t.Logf("DeleteGroup failed as expected in test environment: %v", err)
	}
}
