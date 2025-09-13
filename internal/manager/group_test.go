package manager

import (
	"testing"

	"github.com/nipunap/kim/internal/testutil"
	"github.com/nipunap/kim/pkg/types"

	"github.com/IBM/sarama"
)

// MockGroupClient implements the client interface for group testing
type MockGroupClient struct {
	*testutil.MockClient
	adminClient *MockGroupAdminClient
}

type MockGroupAdminClient struct {
	groups     map[string]*sarama.GroupDescription
	shouldFail bool
}

func (m *MockGroupAdminClient) ListConsumerGroups() (map[string]string, error) {
	if m.shouldFail {
		return nil, sarama.ErrBrokerNotAvailable
	}

	groups := make(map[string]string)
	for groupID := range m.groups {
		groups[groupID] = "consumer"
	}
	return groups, nil
}

func (m *MockGroupAdminClient) DescribeConsumerGroups(groups []string) ([]*sarama.GroupDescription, error) {
	if m.shouldFail {
		return nil, sarama.ErrBrokerNotAvailable
	}

	var result []*sarama.GroupDescription
	for _, groupID := range groups {
		if desc, exists := m.groups[groupID]; exists {
			result = append(result, desc)
		}
	}
	return result, nil
}

func (m *MockGroupAdminClient) DeleteConsumerGroup(groupID string) error {
	if m.shouldFail {
		return sarama.ErrGroupIdNotFound
	}
	delete(m.groups, groupID)
	return nil
}

func newMockGroupClient() *MockGroupClient {
	mockClient := testutil.NewMockClient(testutil.TestProfile(), testutil.TestLogger())

	// Create mock group descriptions
	groups := map[string]*sarama.GroupDescription{
		"test-group-1": {
			GroupId:      "test-group-1",
			State:        "Stable",
			ProtocolType: "consumer",
			Protocol:     "range",
			Members: map[string]*sarama.GroupMemberDescription{
				"member-1": {
					MemberId:   "member-1",
					ClientId:   "client-1",
					ClientHost: "host-1",
				},
				"member-2": {
					MemberId:   "member-2",
					ClientId:   "client-2",
					ClientHost: "host-2",
				},
			},
		},
		"test-group-2": {
			GroupId:      "test-group-2",
			State:        "Empty",
			ProtocolType: "consumer",
			Protocol:     "range",
			Members:      map[string]*sarama.GroupMemberDescription{},
		},
		"test-group-3": {
			GroupId:      "test-group-3",
			State:        "Rebalancing",
			ProtocolType: "consumer",
			Protocol:     "roundrobin",
			Members: map[string]*sarama.GroupMemberDescription{
				"member-1": {
					MemberId:   "member-1",
					ClientId:   "client-1",
					ClientHost: "host-1",
				},
			},
		},
	}

	adminClient := &MockGroupAdminClient{
		groups: groups,
	}

	// Add groups to mock client as well
	mockClient.AddMockGroup("test-group-1", "Stable", "consumer", 2)
	mockClient.AddMockGroup("test-group-2", "Empty", "consumer", 0)
	mockClient.AddMockGroup("test-group-3", "Rebalancing", "consumer", 1)

	return &MockGroupClient{
		MockClient:  mockClient,
		adminClient: adminClient,
	}
}

func TestGroupManagerListGroups(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test basic list
	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 10,
		Format:   "table",
	}

	result, err := gm.ListGroups(opts)
	testutil.AssertNoError(t, err)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if len(result.Groups) == 0 {
		t.Error("Should return groups")
	}

	// Verify pagination
	if result.Pagination == nil {
		t.Error("Pagination should not be nil")
	}

	if result.Pagination.TotalItems != 3 {
		t.Errorf("Expected 3 total items, got %d", result.Pagination.TotalItems)
	}

	// Verify group data
	foundStableGroup := false
	for _, group := range result.Groups {
		if group.GroupID == "test-group-1" {
			foundStableGroup = true
			if group.State != "Stable" {
				t.Errorf("Expected state 'Stable', got '%s'", group.State)
			}
			if group.ProtocolType != "consumer" {
				t.Errorf("Expected protocol type 'consumer', got '%s'", group.ProtocolType)
			}
			if group.MemberCount != 2 {
				t.Errorf("Expected 2 members, got %d", group.MemberCount)
			}
		}
	}

	if !foundStableGroup {
		t.Error("Should find test-group-1 with Stable state")
	}
}

func TestGroupManagerListGroupsWithFilter(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test with filter
	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 10,
		Format:   "table",
		Filter:   "test-group-1",
	}

	result, err := gm.ListGroups(opts)
	testutil.AssertNoError(t, err)

	if len(result.Groups) != 1 {
		t.Errorf("Expected 1 filtered group, got %d", len(result.Groups))
	}

	if result.Groups[0].GroupID != "test-group-1" {
		t.Errorf("Expected group ID 'test-group-1', got '%s'", result.Groups[0].GroupID)
	}
}

func TestGroupManagerListGroupsWithPagination(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test pagination - page 1 with size 2
	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 2,
		Format:   "table",
	}

	result, err := gm.ListGroups(opts)
	testutil.AssertNoError(t, err)

	if len(result.Groups) != 2 {
		t.Errorf("Expected 2 groups on page 1, got %d", len(result.Groups))
	}

	if result.Pagination.CurrentPage != 1 {
		t.Errorf("Expected current page 1, got %d", result.Pagination.CurrentPage)
	}

	if result.Pagination.TotalPages != 2 {
		t.Errorf("Expected 2 total pages, got %d", result.Pagination.TotalPages)
	}

	// Test page 2
	opts.Page = 2
	result, err = gm.ListGroups(opts)
	testutil.AssertNoError(t, err)

	if len(result.Groups) != 1 {
		t.Errorf("Expected 1 group on page 2, got %d", len(result.Groups))
	}
}

func TestGroupManagerDescribeGroup(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test describe existing group
	result, err := gm.DescribeGroup("test-group-1")
	testutil.AssertNoError(t, err)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.GroupID != "test-group-1" {
		t.Errorf("Expected group ID 'test-group-1', got '%s'", result.GroupID)
	}

	if result.State != "Stable" {
		t.Errorf("Expected state 'Stable', got '%s'", result.State)
	}

	if result.ProtocolType != "consumer" {
		t.Errorf("Expected protocol type 'consumer', got '%s'", result.ProtocolType)
	}

	if result.Protocol != "range" {
		t.Errorf("Expected protocol 'range', got '%s'", result.Protocol)
	}

	if len(result.Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(result.Members))
	}

	// Verify member details
	foundMember1 := false
	for _, member := range result.Members {
		if member.MemberID == "member-1" {
			foundMember1 = true
			if member.ClientID != "client-1" {
				t.Errorf("Expected client ID 'client-1', got '%s'", member.ClientID)
			}
			if member.Host != "host-1" {
				t.Errorf("Expected host 'host-1', got '%s'", member.Host)
			}
		}
	}

	if !foundMember1 {
		t.Error("Should find member-1")
	}

	// Test describe non-existent group
	_, err = gm.DescribeGroup("non-existent")
	testutil.AssertError(t, err)
}

func TestGroupManagerDescribeEmptyGroup(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test describe empty group
	result, err := gm.DescribeGroup("test-group-2")
	testutil.AssertNoError(t, err)

	if result.State != "Empty" {
		t.Errorf("Expected state 'Empty', got '%s'", result.State)
	}

	if len(result.Members) != 0 {
		t.Errorf("Expected 0 members for empty group, got %d", len(result.Members))
	}
}

func TestGroupManagerDeleteGroup(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test delete existing group
	err := gm.DeleteGroup("test-group-2")
	testutil.AssertNoError(t, err)

	// Verify group was deleted
	_, err = gm.DescribeGroup("test-group-2")
	testutil.AssertError(t, err)
}

func TestGroupManagerDeleteGroupFailure(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()
	mockClient.adminClient.shouldFail = true

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	err := gm.DeleteGroup("non-existent")
	testutil.AssertError(t, err)
}

func TestGroupManagerResetOffsets(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test reset offsets (this is a simplified implementation)
	err := gm.ResetOffsets("test-group-1", "test-topic", "earliest")

	// Since this is a mock implementation, we expect it to succeed
	// In a real implementation, this would interact with Kafka to reset offsets
	testutil.AssertNoError(t, err)
}

func TestGroupManagerSorting(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test sort by group ID ascending
	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 10,
		Format:   "table",
		SortBy:   "group",
		SortDesc: false,
	}

	result, err := gm.ListGroups(opts)
	testutil.AssertNoError(t, err)

	// Verify groups are sorted by ID
	for i := 1; i < len(result.Groups); i++ {
		if result.Groups[i-1].GroupID > result.Groups[i].GroupID {
			t.Error("Groups should be sorted by ID ascending")
		}
	}

	// Test sort by state descending
	opts.SortBy = "state"
	opts.SortDesc = true

	result, err = gm.ListGroups(opts)
	testutil.AssertNoError(t, err)

	// Verify groups are sorted by state descending
	for i := 1; i < len(result.Groups); i++ {
		if result.Groups[i-1].State < result.Groups[i].State {
			t.Error("Groups should be sorted by state descending")
		}
	}
}

func TestGroupManagerNotConnected(t *testing.T) {
	mockClient := newMockGroupClient()
	// Don't connect

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 10,
		Format:   "table",
	}

	_, err := gm.ListGroups(opts)
	testutil.AssertError(t, err)
}

func TestGroupManagerInvalidRequests(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test delete group with empty name
	err := gm.DeleteGroup("")
	testutil.AssertError(t, err)

	// Test describe group with empty name
	_, err = gm.DescribeGroup("")
	testutil.AssertError(t, err)

	// Test reset offsets with empty group
	err = gm.ResetOffsets("", "topic", "earliest")
	testutil.AssertError(t, err)

	// Test reset offsets with empty topic
	err = gm.ResetOffsets("group", "", "earliest")
	testutil.AssertError(t, err)

	// Test reset offsets with invalid strategy
	err = gm.ResetOffsets("group", "topic", "invalid")
	testutil.AssertError(t, err)
}

func TestGroupManagerFilterByState(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test filter by state
	opts := &types.DisplayOptions{
		Page:     1,
		PageSize: 10,
		Format:   "table",
		Filter:   "Stable",
	}

	result, err := gm.ListGroups(opts)
	testutil.AssertNoError(t, err)

	// Should find groups with "Stable" in their state or ID
	foundStable := false
	for _, group := range result.Groups {
		if group.State == "Stable" || group.GroupID == "test-group-1" {
			foundStable = true
		}
	}

	if !foundStable {
		t.Error("Should find stable group when filtering by 'Stable'")
	}
}

func TestGroupManagerCalculateLag(t *testing.T) {
	mockClient := newMockGroupClient()
	mockClient.Connect()

	gm := NewGroupManager(mockClient, testutil.TestLogger())

	// Test describe group and verify lag calculation
	result, err := gm.DescribeGroup("test-group-1")
	testutil.AssertNoError(t, err)

	// In our mock implementation, lag is set to 0
	// In a real implementation, this would calculate actual lag
	for _, member := range result.Members {
		if member.TotalLag < 0 {
			t.Errorf("Lag should not be negative, got %d", member.TotalLag)
		}
	}
}
