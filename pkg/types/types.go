package types

import (
	"fmt"
	"time"
)

// Pagination represents pagination information
type Pagination struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	PageSize    int `json:"page_size"`
	TotalItems  int `json:"total_items"`
}

// ListOptions represents common listing options
type ListOptions struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Pattern  string `json:"pattern,omitempty"`
	SortBy   string `json:"sort_by"`
	Order    string `json:"order"` // "asc" or "desc"
}

// Topic-related types

// TopicInfo represents basic topic information
type TopicInfo struct {
	Name              string `json:"name"`
	Partitions        int32  `json:"partitions"`
	ReplicationFactor int32  `json:"replication_factor"`
	Internal          bool   `json:"internal"`
}

// TopicList represents a paginated list of topics
type TopicList struct {
	Topics     []*TopicInfo `json:"topics"`
	Pagination *Pagination  `json:"pagination"`
}

// PartitionInfo represents partition details
type PartitionInfo struct {
	ID              int32   `json:"id"`
	Leader          int32   `json:"leader"`
	Replicas        []int32 `json:"replicas"`
	InSyncReplicas  []int32 `json:"in_sync_replicas"`
	OfflineReplicas []int32 `json:"offline_replicas"`
}

// TopicDetails represents detailed topic information
type TopicDetails struct {
	Name              string            `json:"name"`
	Partitions        int32             `json:"partitions"`
	ReplicationFactor int32             `json:"replication_factor"`
	Internal          bool              `json:"internal"`
	Configs           map[string]string `json:"configs"`
	PartitionDetails  []*PartitionInfo  `json:"partition_details"`
}

// CreateTopicRequest represents a request to create a topic
type CreateTopicRequest struct {
	Name              string            `json:"name"`
	Partitions        int32             `json:"partitions"`
	ReplicationFactor int16             `json:"replication_factor"`
	Configs           map[string]string `json:"configs,omitempty"`
}

// Consumer Group related types

// GroupInfo represents basic consumer group information
type GroupInfo struct {
	GroupID      string `json:"group_id"`
	State        string `json:"state"`
	ProtocolType string `json:"protocol_type"`
	MemberCount  int    `json:"member_count"`
}

// GroupList represents a paginated list of consumer groups
type GroupList struct {
	Groups     []*GroupInfo `json:"groups"`
	Pagination *Pagination  `json:"pagination"`
}

// CoordinatorInfo represents coordinator information
type CoordinatorInfo struct {
	ID   int32  `json:"id"`
	Host string `json:"host"`
	Port int32  `json:"port"`
}

// PartitionAssignment represents a partition assignment
type PartitionAssignment struct {
	Topic         string `json:"topic"`
	Partition     int32  `json:"partition"`
	CurrentOffset int64  `json:"current_offset"`
	LogEndOffset  int64  `json:"log_end_offset"`
	Lag           int64  `json:"lag"`
}

// MemberInfo represents consumer group member information
type MemberInfo struct {
	MemberID           string                 `json:"member_id"`
	ClientID           string                 `json:"client_id"`
	Host               string                 `json:"host"`
	AssignedPartitions []*PartitionAssignment `json:"assigned_partitions"`
	TotalLag           int64                  `json:"total_lag"`
}

// GroupDetails represents detailed consumer group information
type GroupDetails struct {
	GroupID      string           `json:"group_id"`
	State        string           `json:"state"`
	ProtocolType string           `json:"protocol_type"`
	Protocol     string           `json:"protocol"`
	Coordinator  *CoordinatorInfo `json:"coordinator"`
	Members      []*MemberInfo    `json:"members"`
	TotalLag     int64            `json:"total_lag"`
}

// ResetOffsetsRequest represents a request to reset consumer group offsets
type ResetOffsetsRequest struct {
	GroupID    string     `json:"group_id"`
	Topics     []string   `json:"topics,omitempty"`
	ToOffset   *int64     `json:"to_offset,omitempty"`
	ToEarliest bool       `json:"to_earliest,omitempty"`
	ToLatest   bool       `json:"to_latest,omitempty"`
	ToDateTime *time.Time `json:"to_datetime,omitempty"`
}

// Message related types

// Message represents a Kafka message
type Message struct {
	Topic     string            `json:"topic"`
	Partition int32             `json:"partition"`
	Offset    int64             `json:"offset"`
	Timestamp time.Time         `json:"timestamp"`
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Headers   map[string]string `json:"headers"`
}

// MessageList represents a paginated list of messages
type MessageList struct {
	Messages   []*Message  `json:"messages"`
	Pagination *Pagination `json:"pagination"`
}

// ProduceRequest represents a request to produce a message
type ProduceRequest struct {
	Topic     string            `json:"topic"`
	Key       string            `json:"key,omitempty"`
	Value     string            `json:"value"`
	Partition *int32            `json:"partition,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// ProduceResponse represents the response from producing a message
type ProduceResponse struct {
	Topic     string    `json:"topic"`
	Partition int32     `json:"partition"`
	Offset    int64     `json:"offset"`
	Timestamp time.Time `json:"timestamp"`
}

// ConsumeRequest represents a request to start consuming messages
type ConsumeRequest struct {
	Topic         string `json:"topic"`
	Partition     int32  `json:"partition"`
	GroupID       string `json:"group_id"`
	FromBeginning bool   `json:"from_beginning"`
}

// ConsumerInfo represents information about an active consumer
type ConsumerInfo struct {
	Topic         string `json:"topic"`
	Partition     int32  `json:"partition"`
	GroupID       string `json:"group_id"`
	FromBeginning bool   `json:"from_beginning"`
}

// GetMessagesRequest represents a request to get messages from a topic
type GetMessagesRequest struct {
	Topic         string `json:"topic"`
	Partition     int32  `json:"partition"`
	FromBeginning bool   `json:"from_beginning"`
	Limit         int    `json:"limit"`
	Offset        *int64 `json:"offset,omitempty"`
}

// Profile related types

// ProfileInfo represents profile information for display
type ProfileInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Details string `json:"details"`
	Active  bool   `json:"active"`
}

// UI related types

// DisplayOptions represents display formatting options
type DisplayOptions struct {
	Format      string `json:"format"`       // "table", "json", "yaml"
	ColorScheme string `json:"color_scheme"` // "default", "dark", "light"
	NoHeaders   bool   `json:"no_headers"`
	Compact     bool   `json:"compact"`
}

// InteractiveState represents the state of interactive mode
type InteractiveState struct {
	CurrentView   string `json:"current_view"`   // "topics", "groups", "messages", etc.
	SelectedItem  string `json:"selected_item"`  // Currently selected item
	ScrollOffset  int    `json:"scroll_offset"`  // Current scroll position
	SearchPattern string `json:"search_pattern"` // Current search pattern
	CommandMode   bool   `json:"command_mode"`   // Whether in command mode
	SearchMode    bool   `json:"search_mode"`    // Whether in search mode
}

// Command represents a command in interactive mode
type Command struct {
	Name        string            `json:"name"`
	Args        []string          `json:"args"`
	Flags       map[string]string `json:"flags"`
	Description string            `json:"description"`
}

// Error types

// KimError represents an application error
type KimError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *KimError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewKimError creates a new KimError
func NewKimError(code, message string) *KimError {
	return &KimError{
		Code:    code,
		Message: message,
	}
}

// NewKimErrorWithDetails creates a new KimError with details
func NewKimErrorWithDetails(code, message, details string) *KimError {
	return &KimError{
		Code:    code,
		Message: message,
		Details: details,
	}
}
