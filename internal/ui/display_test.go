package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"kim/pkg/types"
)

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestDisplayTopicList(t *testing.T) {
	topicList := &types.TopicList{
		Topics: []*types.TopicInfo{
			{
				Name:              "test-topic-1",
				Partitions:        3,
				ReplicationFactor: 2,
			},
			{
				Name:              "test-topic-2",
				Partitions:        1,
				ReplicationFactor: 1,
			},
		},
		Pagination: &types.Pagination{
			CurrentPage: 1,
			TotalPages:  1,
			TotalItems:  2,
			PageSize:    10,
		},
	}

	// Test table format
	opts := &types.DisplayOptions{Format: "table"}
	output := captureOutput(func() {
		err := DisplayTopicList(topicList, opts)
		if err != nil {
			t.Errorf("DisplayTopicList failed: %v", err)
		}
	})

	if !strings.Contains(output, "test-topic-1") {
		t.Error("Output should contain test-topic-1")
	}
	if !strings.Contains(output, "test-topic-2") {
		t.Error("Output should contain test-topic-2")
	}

	// Test JSON format
	opts.Format = "json"
	output = captureOutput(func() {
		err := DisplayTopicList(topicList, opts)
		if err != nil {
			t.Errorf("DisplayTopicList JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, "test-topic-1") {
		t.Error("JSON output should contain topic name")
	}
}

func TestDisplayTopicDetails(t *testing.T) {
	details := &types.TopicDetails{
		Name:              "test-topic",
		Partitions:        2,
		ReplicationFactor: 1,
		PartitionDetails: []*types.PartitionInfo{
			{
				ID:              0,
				Leader:          1,
				Replicas:        []int32{1, 2},
				InSyncReplicas:  []int32{1, 2},
				OfflineReplicas: []int32{},
			},
		},
		Configs: map[string]string{
			"retention.ms": "604800000",
		},
	}

	// Test table format
	opts := &types.DisplayOptions{Format: "table"}
	output := captureOutput(func() {
		err := DisplayTopicDetails(details, opts)
		if err != nil {
			t.Errorf("DisplayTopicDetails failed: %v", err)
		}
	})

	if !strings.Contains(output, "test-topic") {
		t.Error("Output should contain topic name")
	}
}

func TestDisplayGroupList(t *testing.T) {
	groupList := &types.GroupList{
		Groups: []*types.GroupInfo{
			{
				GroupID:      "group-1",
				State:        "Stable",
				ProtocolType: "consumer",
				MemberCount:  2,
			},
		},
		Pagination: &types.Pagination{
			CurrentPage: 1,
			TotalPages:  1,
			TotalItems:  1,
			PageSize:    10,
		},
	}

	// Test table format
	opts := &types.DisplayOptions{Format: "table"}
	output := captureOutput(func() {
		err := DisplayGroupList(groupList, opts)
		if err != nil {
			t.Errorf("DisplayGroupList failed: %v", err)
		}
	})

	if !strings.Contains(output, "group-1") {
		t.Error("Output should contain group-1")
	}
}

func TestDisplayGroupDetails(t *testing.T) {
	details := &types.GroupDetails{
		GroupID:      "test-group",
		State:        "Stable",
		ProtocolType: "consumer",
		Protocol:     "range",
		Members: []*types.MemberInfo{
			{
				MemberID: "member-1",
				ClientID: "client-1",
				Host:     "host-1",
				TotalLag: 100,
			},
		},
		Coordinator: &types.CoordinatorInfo{
			ID:   1,
			Host: "broker-1",
			Port: 9092,
		},
	}

	// Test table format
	opts := &types.DisplayOptions{Format: "table"}
	output := captureOutput(func() {
		err := DisplayGroupDetails(details, opts)
		if err != nil {
			t.Errorf("DisplayGroupDetails failed: %v", err)
		}
	})

	if !strings.Contains(output, "test-group") {
		t.Error("Output should contain group ID")
	}
}

func TestDisplayMessage(t *testing.T) {
	message := &types.Message{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1234,
		Key:       "test-key",
		Value:     "test-value",
		Headers: map[string]string{
			"header1": "value1",
		},
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	// Test table format
	opts := &types.DisplayOptions{Format: "table"}
	output := captureOutput(func() {
		err := DisplayMessage(message, opts)
		if err != nil {
			t.Errorf("DisplayMessage failed: %v", err)
		}
	})

	if !strings.Contains(output, "test-topic") {
		t.Error("Output should contain topic name")
	}
	if !strings.Contains(output, "test-key") {
		t.Error("Output should contain message key")
	}
	if !strings.Contains(output, "test-value") {
		t.Error("Output should contain message value")
	}
}

func TestDisplayProfileList(t *testing.T) {
	profiles := []*types.ProfileInfo{
		{
			Name:    "kafka-local",
			Type:    "kafka",
			Details: "localhost:9092",
			Active:  true,
		},
		{
			Name:    "msk-prod",
			Type:    "msk",
			Details: "us-east-1",
			Active:  false,
		},
	}

	// Test table format
	opts := &types.DisplayOptions{Format: "table"}
	output := captureOutput(func() {
		err := DisplayProfileList(profiles, opts)
		if err != nil {
			t.Errorf("DisplayProfileList failed: %v", err)
		}
	})

	if !strings.Contains(output, "kafka-local") {
		t.Error("Output should contain profile name")
	}
	if !strings.Contains(output, "msk-prod") {
		t.Error("Output should contain profile name")
	}
}

func TestDisplayInvalidFormat(t *testing.T) {
	topicList := &types.TopicList{
		Topics: []*types.TopicInfo{
			{Name: "test", Partitions: 1, ReplicationFactor: 1},
		},
	}

	opts := &types.DisplayOptions{Format: "invalid"}
	err := DisplayTopicList(topicList, opts)
	if err == nil {
		t.Error("Should return error for invalid format")
	}
}

func TestDisplayEmptyData(t *testing.T) {
	// Test empty topic list
	emptyTopicList := &types.TopicList{
		Topics: []*types.TopicInfo{},
		Pagination: &types.Pagination{
			CurrentPage: 1,
			TotalPages:  0,
			TotalItems:  0,
			PageSize:    10,
		},
	}

	opts := &types.DisplayOptions{Format: "table"}
	output := captureOutput(func() {
		err := DisplayTopicList(emptyTopicList, opts)
		if err != nil {
			t.Errorf("DisplayTopicList failed: %v", err)
		}
	})

	// Should handle empty list gracefully
	if len(output) == 0 {
		t.Error("Should produce some output even for empty list")
	}
}

func TestFormatInt32Slice(t *testing.T) {
	// Test empty slice
	result := formatInt32Slice([]int32{})
	if result != "[]" {
		t.Errorf("Expected '[]', got '%s'", result)
	}

	// Test single element
	result = formatInt32Slice([]int32{1})
	if result != "[1]" {
		t.Errorf("Expected '[1]', got '%s'", result)
	}

	// Test multiple elements
	result = formatInt32Slice([]int32{1, 2, 3})
	if result != "[1,2,3]" {
		t.Errorf("Expected '[1,2,3]', got '%s'", result)
	}
}

func TestDisplayNilData(t *testing.T) {
	opts := &types.DisplayOptions{Format: "table"}

	// Test with nil topic list
	err := DisplayTopicList(nil, opts)
	if err == nil {
		t.Error("Should return error for nil topic list")
	}

	// Test with nil group list
	err = DisplayGroupList(nil, opts)
	if err == nil {
		t.Error("Should return error for nil group list")
	}

	// Test with nil message
	err = DisplayMessage(nil, opts)
	if err == nil {
		t.Error("Should return error for nil message")
	}

	// Test with nil profile list
	err = DisplayProfileList(nil, opts)
	if err == nil {
		t.Error("Should return error for nil profile list")
	}
}
