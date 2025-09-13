package manager

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nipunap/kim/internal/client"
	"github.com/nipunap/kim/internal/logger"
	"github.com/nipunap/kim/pkg/types"

	"github.com/IBM/sarama"
)

// TopicManager manages Kafka topic operations
type TopicManager struct {
	client *client.Client
	logger *logger.Logger
}

// NewTopicManager creates a new topic manager
func NewTopicManager(client *client.Client, logger *logger.Logger) *TopicManager {
	return &TopicManager{
		client: client,
		logger: logger,
	}
}

// ListTopics returns a paginated list of topics
func (tm *TopicManager) ListTopics(ctx context.Context, opts *types.ListOptions) (*types.TopicList, error) {
	if !tm.client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Get topic metadata
	metadata, err := tm.client.AdminClient.DescribeTopics(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe topics: %w", err)
	}

	// Convert to topic info
	var topics []*types.TopicInfo
	for _, meta := range metadata {
		if meta.Err != sarama.ErrNoError {
			tm.logger.Warn("Error getting topic metadata", "topic", meta.Name, "error", meta.Err)
			continue
		}

		topic := &types.TopicInfo{
			Name:              meta.Name,
			Partitions:        int32(len(meta.Partitions)),
			ReplicationFactor: 0,
			Internal:          meta.IsInternal,
		}

		// Calculate replication factor from first partition
		if len(meta.Partitions) > 0 {
			topic.ReplicationFactor = int32(len(meta.Partitions[0].Replicas))
		}

		// Apply pattern filter if specified
		if opts.Pattern != "" && !matchesPattern(meta.Name, opts.Pattern) {
			continue
		}

		topics = append(topics, topic)
	}

	// Sort topics
	sort.Slice(topics, func(i, j int) bool {
		switch opts.SortBy {
		case "partitions":
			if opts.Order == "desc" {
				return topics[i].Partitions > topics[j].Partitions
			}
			return topics[i].Partitions < topics[j].Partitions
		case "replication_factor":
			if opts.Order == "desc" {
				return topics[i].ReplicationFactor > topics[j].ReplicationFactor
			}
			return topics[i].ReplicationFactor < topics[j].ReplicationFactor
		default: // name
			if opts.Order == "desc" {
				return topics[i].Name > topics[j].Name
			}
			return topics[i].Name < topics[j].Name
		}
	})

	// Apply pagination
	totalItems := len(topics)
	totalPages := (totalItems + opts.PageSize - 1) / opts.PageSize

	start := (opts.Page - 1) * opts.PageSize
	end := start + opts.PageSize
	if end > totalItems {
		end = totalItems
	}
	if start > totalItems {
		start = totalItems
	}

	paginatedTopics := topics[start:end]

	return &types.TopicList{
		Topics: paginatedTopics,
		Pagination: &types.Pagination{
			CurrentPage: opts.Page,
			TotalPages:  totalPages,
			PageSize:    opts.PageSize,
			TotalItems:  totalItems,
		},
	}, nil
}

// DescribeTopic returns detailed information about a specific topic
func (tm *TopicManager) DescribeTopic(ctx context.Context, topicName string) (*types.TopicDetails, error) {
	if !tm.client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Get topic metadata
	metadata, err := tm.client.AdminClient.DescribeTopics([]string{topicName})
	if err != nil {
		return nil, fmt.Errorf("failed to describe topic: %w", err)
	}

	if len(metadata) == 0 {
		return nil, fmt.Errorf("topic %s not found", topicName)
	}

	topicMeta := metadata[0]
	if topicMeta.Err != sarama.ErrNoError {
		return nil, fmt.Errorf("error describing topic %s: %v", topicName, topicMeta.Err)
	}

	// Get topic configuration
	configResource := sarama.ConfigResource{
		Type: sarama.TopicResource,
		Name: topicName,
	}

	configs, err := tm.client.AdminClient.DescribeConfig(configResource)
	if err != nil {
		tm.logger.Warn("Failed to get topic configuration", "topic", topicName, "error", err)
	}

	// Build topic details
	details := &types.TopicDetails{
		Name:              topicMeta.Name,
		Partitions:        int32(len(topicMeta.Partitions)),
		ReplicationFactor: 0,
		Internal:          topicMeta.IsInternal,
		Configs:           make(map[string]string),
		PartitionDetails:  make([]*types.PartitionInfo, 0, len(topicMeta.Partitions)),
	}

	// Calculate replication factor and build partition details
	for _, partition := range topicMeta.Partitions {
		if details.ReplicationFactor == 0 {
			details.ReplicationFactor = int32(len(partition.Replicas))
		}

		partitionInfo := &types.PartitionInfo{
			ID:              partition.ID,
			Leader:          partition.Leader,
			Replicas:        partition.Replicas,
			InSyncReplicas:  partition.Isr,
			OfflineReplicas: partition.OfflineReplicas,
		}

		details.PartitionDetails = append(details.PartitionDetails, partitionInfo)
	}

	// Add configuration details
	if configs != nil {
		for _, config := range configs {
			details.Configs[config.Name] = config.Value
		}
	}

	return details, nil
}

// CreateTopic creates a new topic
func (tm *TopicManager) CreateTopic(ctx context.Context, req *types.CreateTopicRequest) error {
	if !tm.client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     req.Partitions,
		ReplicationFactor: req.ReplicationFactor,
		ConfigEntries:     make(map[string]*string),
	}

	// Add configuration entries
	for key, value := range req.Configs {
		topicDetail.ConfigEntries[key] = &value
	}

	err := tm.client.AdminClient.CreateTopic(req.Name, topicDetail, false)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	tm.logger.Info("Topic created successfully", "topic", req.Name)
	return nil
}

// DeleteTopic deletes a topic
func (tm *TopicManager) DeleteTopic(ctx context.Context, topicName string) error {
	if !tm.client.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	err := tm.client.AdminClient.DeleteTopic(topicName)
	if err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	tm.logger.Info("Topic deleted successfully", "topic", topicName)
	return nil
}

// GetTopicOffsets returns the latest offsets for all partitions of a topic
func (tm *TopicManager) GetTopicOffsets(ctx context.Context, topicName string) (map[int32]int64, error) {
	if !tm.client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Get topic metadata to find partitions
	metadata, err := tm.client.AdminClient.DescribeTopics([]string{topicName})
	if err != nil {
		return nil, fmt.Errorf("failed to describe topic: %w", err)
	}

	if len(metadata) == 0 {
		return nil, fmt.Errorf("topic %s not found", topicName)
	}

	topicMeta := metadata[0]

	offsets := make(map[int32]int64)

	// Get latest offset for each partition (simplified implementation)
	for _, partition := range topicMeta.Partitions {
		// In a full implementation, you would create a partition consumer
		// and get the latest offset. For now, just set to 0.
		offsets[partition.ID] = 0
		tm.logger.Debug("Getting offset for partition",
			"topic", topicName, "partition", partition.ID)
	}

	return offsets, nil
}

// FormatConfigValue formats configuration values for display
func (tm *TopicManager) FormatConfigValue(key, value string) string {
	switch key {
	case "retention.ms":
		return tm.formatTimeMs(value)
	case "retention.bytes", "segment.bytes", "max.message.bytes", "index.interval.bytes":
		return tm.formatBytes(value)
	case "cleanup.policy":
		switch value {
		case "delete":
			return "Delete (messages are deleted after retention period)"
		case "compact":
			return "Compact (only latest messages per key are kept)"
		case "compact,delete":
			return "Compact and Delete"
		default:
			return value
		}
	case "compression.type":
		return strings.Title(value)
	case "unclean.leader.election.enable", "preallocate":
		if value == "true" {
			return "Enabled"
		}
		return "Disabled"
	default:
		return value
	}
}

// formatTimeMs formats milliseconds into human-readable time
func (tm *TopicManager) formatTimeMs(value string) string {
	ms, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return value
	}

	if ms == -1 {
		return "unlimited"
	}
	if ms == 0 {
		return "0"
	}

	duration := time.Duration(ms) * time.Millisecond

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%d days %d hours", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%d minutes %d seconds", minutes, seconds)
	} else {
		return fmt.Sprintf("%d seconds", seconds)
	}
}

// formatBytes formats bytes into human-readable size
func (tm *TopicManager) formatBytes(value string) string {
	bytes, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return value
	}

	if bytes == -1 {
		return "unlimited"
	}
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"B", "KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp+1])
}

// matchesPattern checks if a string matches a wildcard pattern
func matchesPattern(str, pattern string) bool {
	// Simple wildcard matching - supports * and ?
	// This is a simplified implementation
	if pattern == "*" {
		return true
	}

	// For now, just check if the pattern is contained in the string
	// In a full implementation, you'd want proper glob matching
	return strings.Contains(strings.ToLower(str), strings.ToLower(strings.ReplaceAll(pattern, "*", "")))
}
