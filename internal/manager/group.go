package manager

import (
	"context"
	"fmt"
	"sort"

	"github.com/nipunap/kim/internal/client"
	"github.com/nipunap/kim/internal/logger"
	"github.com/nipunap/kim/pkg/types"

	"github.com/IBM/sarama"
)

// GroupManager manages Kafka consumer group operations
type GroupManager struct {
	client *client.Client
	logger *logger.Logger
}

// NewGroupManager creates a new group manager
func NewGroupManager(client *client.Client, logger *logger.Logger) *GroupManager {
	return &GroupManager{
		client: client,
		logger: logger,
	}
}

// ListGroups returns a paginated list of consumer groups
func (gm *GroupManager) ListGroups(ctx context.Context, opts *types.ListOptions) (*types.GroupList, error) {
	if !gm.client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Get consumer group list
	groupList, err := gm.client.AdminClient.ListConsumerGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to list consumer groups: %w", err)
	}

	// Convert to group info
	var groups []*types.GroupInfo
	for groupID, groupType := range groupList {
		// Apply pattern filter if specified
		if opts.Pattern != "" && !matchesPattern(groupID, opts.Pattern) {
			continue
		}

		group := &types.GroupInfo{
			GroupID:      groupID,
			ProtocolType: groupType,
			State:        "Unknown", // We'll need to describe the group to get the state
		}

		groups = append(groups, group)
	}

	// Sort groups
	sort.Slice(groups, func(i, j int) bool {
		switch opts.SortBy {
		case "state":
			if opts.Order == "desc" {
				return groups[i].State > groups[j].State
			}
			return groups[i].State < groups[j].State
		case "protocol_type":
			if opts.Order == "desc" {
				return groups[i].ProtocolType > groups[j].ProtocolType
			}
			return groups[i].ProtocolType < groups[j].ProtocolType
		default: // group_id
			if opts.Order == "desc" {
				return groups[i].GroupID > groups[j].GroupID
			}
			return groups[i].GroupID < groups[j].GroupID
		}
	})

	// Apply pagination
	totalItems := len(groups)
	totalPages := (totalItems + opts.PageSize - 1) / opts.PageSize

	start := (opts.Page - 1) * opts.PageSize
	end := start + opts.PageSize
	if end > totalItems {
		end = totalItems
	}
	if start > totalItems {
		start = totalItems
	}

	paginatedGroups := groups[start:end]

	return &types.GroupList{
		Groups: paginatedGroups,
		Pagination: &types.Pagination{
			CurrentPage: opts.Page,
			TotalPages:  totalPages,
			PageSize:    opts.PageSize,
			TotalItems:  totalItems,
		},
	}, nil
}

// DescribeGroup returns detailed information about a specific consumer group
func (gm *GroupManager) DescribeGroup(ctx context.Context, groupID string) (*types.GroupDetails, error) {
	if !gm.client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Describe the consumer group
	groupDescriptions, err := gm.client.AdminClient.DescribeConsumerGroups([]string{groupID})
	if err != nil {
		return nil, fmt.Errorf("failed to describe consumer group: %w", err)
	}

	if len(groupDescriptions) == 0 {
		return nil, fmt.Errorf("consumer group %s not found", groupID)
	}

	groupDesc := groupDescriptions[0]
	if groupDesc.Err != sarama.ErrNoError {
		return nil, fmt.Errorf("error describing consumer group %s: %v", groupID, groupDesc.Err)
	}

	// Build group details
	details := &types.GroupDetails{
		GroupID:      groupID,
		State:        groupDesc.State,
		ProtocolType: groupDesc.ProtocolType,
		Protocol:     groupDesc.Protocol,
		Members:      make([]*types.MemberInfo, 0, len(groupDesc.Members)),
	}

	// Add coordinator information (simplified for now)
	details.Coordinator = &types.CoordinatorInfo{
		ID:   -1,
		Host: "N/A",
		Port: -1,
	}

	// Process members
	for memberID, member := range groupDesc.Members {
		memberInfo := &types.MemberInfo{
			MemberID: memberID,
			ClientID: member.ClientId,
			Host:     member.ClientHost,
		}

		// Parse member assignment to get topic partitions
		if len(member.MemberAssignment) > 0 {
			assignment, err := member.GetMemberAssignment()
			if err != nil {
				gm.logger.Warn("Failed to parse member assignment",
					"group", groupID, "member", memberID, "error", err)
			} else {
				for topic, partitions := range assignment.Topics {
					for _, partition := range partitions {
						memberInfo.AssignedPartitions = append(memberInfo.AssignedPartitions, &types.PartitionAssignment{
							Topic:     topic,
							Partition: partition,
						})
					}
				}
			}
		}

		details.Members = append(details.Members, memberInfo)
	}

	// Get consumer group offsets for lag calculation
	if err := gm.calculateLag(ctx, details); err != nil {
		gm.logger.Warn("Failed to calculate consumer lag", "group", groupID, "error", err)
	}

	return details, nil
}

// calculateLag calculates the lag for each partition assignment
func (gm *GroupManager) calculateLag(ctx context.Context, details *types.GroupDetails) error {
	// Simplified implementation - just set lag to 0 for now
	// In a full implementation, you would need to:
	// 1. Get the coordinator for the consumer group
	// 2. Fetch consumer offsets for all assigned partitions
	// 3. Get the latest offsets for comparison
	// 4. Calculate the difference

	for _, member := range details.Members {
		for _, assignment := range member.AssignedPartitions {
			assignment.CurrentOffset = 0
			assignment.LogEndOffset = 0
			assignment.Lag = 0
		}
		member.TotalLag = 0
	}
	details.TotalLag = 0

	return nil
}

// ResetGroupOffsets resets consumer group offsets for specified topics/partitions
func (gm *GroupManager) ResetGroupOffsets(ctx context.Context, req *types.ResetOffsetsRequest) error {
	if !gm.client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	// This would require implementing offset reset functionality
	// For now, return an error indicating it's not implemented
	return fmt.Errorf("reset group offsets not implemented yet")
}

// DeleteGroup deletes a consumer group
func (gm *GroupManager) DeleteGroup(ctx context.Context, groupID string) error {
	if !gm.client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	// Delete the consumer group
	err := gm.client.AdminClient.DeleteConsumerGroup(groupID)
	if err != nil {
		return fmt.Errorf("failed to delete consumer group: %w", err)
	}

	gm.logger.Info("Consumer group deleted successfully", "group", groupID)
	return nil
}
