package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nipunap/kim/pkg/types"

	"gopkg.in/yaml.v3"
)

// DisplayTopicList displays a list of topics
func DisplayTopicList(topicList *types.TopicList, opts *types.DisplayOptions) error {
	if topicList == nil {
		return fmt.Errorf("topic list cannot be nil")
	}
	switch opts.Format {
	case "json":
		return displayJSON(topicList)
	case "yaml":
		return displayYAML(topicList)
	case "table", "":
		return displayTopicTable(topicList)
	default:
		return fmt.Errorf("invalid format: %s", opts.Format)
	}
}

// DisplayTopicDetails displays detailed topic information
func DisplayTopicDetails(details *types.TopicDetails, opts *types.DisplayOptions) error {
	switch opts.Format {
	case "json":
		return displayJSON(details)
	case "yaml":
		return displayYAML(details)
	default:
		return displayTopicDetailsTable(details)
	}
}

// DisplayGroupList displays a list of consumer groups
func DisplayGroupList(groupList *types.GroupList, opts *types.DisplayOptions) error {
	if groupList == nil {
		return fmt.Errorf("group list cannot be nil")
	}
	switch opts.Format {
	case "json":
		return displayJSON(groupList)
	case "yaml":
		return displayYAML(groupList)
	default:
		return displayGroupTable(groupList)
	}
}

// DisplayGroupDetails displays detailed consumer group information
func DisplayGroupDetails(details *types.GroupDetails, opts *types.DisplayOptions) error {
	switch opts.Format {
	case "json":
		return displayJSON(details)
	case "yaml":
		return displayYAML(details)
	default:
		return displayGroupDetailsTable(details)
	}
}

// DisplayMessage displays a single message
func DisplayMessage(message *types.Message, opts *types.DisplayOptions) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}
	switch opts.Format {
	case "json":
		return displayJSON(message)
	case "yaml":
		return displayYAML(message)
	case "table", "":
		return displayMessageTable(message)
	default:
		return fmt.Errorf("invalid format: %s", opts.Format)
	}
}

// DisplayProduceResponse displays the response from producing a message
func DisplayProduceResponse(response *types.ProduceResponse, opts *types.DisplayOptions) error {
	if response == nil {
		return fmt.Errorf("produce response cannot be nil")
	}
	switch opts.Format {
	case "json":
		return displayJSON(response)
	case "yaml":
		return displayYAML(response)
	case "table", "":
		return displayProduceResponseTable(response)
	default:
		return fmt.Errorf("invalid format: %s", opts.Format)
	}
}

// DisplayProfileList displays a list of profiles
func DisplayProfileList(profiles []*types.ProfileInfo, opts *types.DisplayOptions) error {
	if profiles == nil {
		return fmt.Errorf("profiles cannot be nil")
	}
	switch opts.Format {
	case "json":
		return displayJSON(profiles)
	case "yaml":
		return displayYAML(profiles)
	case "table", "":
		return displayProfileTable(profiles)
	default:
		return fmt.Errorf("invalid format: %s", opts.Format)
	}
}

// displayJSON displays data as JSON
func displayJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// displayYAML displays data as YAML
func displayYAML(data interface{}) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(data)
}

// displayTopicTable displays topics in table format
func displayTopicTable(topicList *types.TopicList) error {
	if len(topicList.Topics) == 0 {
		fmt.Println("No topics found")
		return nil
	}

	// Print header
	fmt.Printf("%-50s %-12s %-20s %-10s\n", "TOPIC NAME", "PARTITIONS", "REPLICATION FACTOR", "INTERNAL")
	fmt.Println(strings.Repeat("-", 92))

	// Print topics
	for _, topic := range topicList.Topics {
		internal := "false"
		if topic.Internal {
			internal = "true"
		}
		fmt.Printf("%-50s %-12d %-20d %-10s\n",
			topic.Name, topic.Partitions, topic.ReplicationFactor, internal)
	}

	// Print pagination info
	if topicList.Pagination != nil {
		fmt.Printf("\nPage %d of %d (%d total topics)\n",
			topicList.Pagination.CurrentPage,
			topicList.Pagination.TotalPages,
			topicList.Pagination.TotalItems)
	}

	return nil
}

// displayTopicDetailsTable displays topic details in table format
func displayTopicDetailsTable(details *types.TopicDetails) error {
	fmt.Printf("Topic: %s\n", details.Name)
	fmt.Println(strings.Repeat("=", 50))

	// Basic information
	fmt.Printf("Partitions: %d\n", details.Partitions)
	fmt.Printf("Replication Factor: %d\n", details.ReplicationFactor)
	fmt.Printf("Internal: %t\n", details.Internal)
	fmt.Println()

	// Partition details
	if len(details.PartitionDetails) > 0 {
		fmt.Println("Partition Details:")
		fmt.Printf("%-10s %-8s %-20s %-20s %-20s\n", "PARTITION", "LEADER", "REPLICAS", "IN-SYNC", "OFFLINE")
		fmt.Println(strings.Repeat("-", 78))

		for _, partition := range details.PartitionDetails {
			fmt.Printf("%-10d %-8d %-20s %-20s %-20s\n",
				partition.ID,
				partition.Leader,
				formatInt32Slice(partition.Replicas),
				formatInt32Slice(partition.InSyncReplicas),
				formatInt32Slice(partition.OfflineReplicas))
		}
		fmt.Println()
	}

	// Configuration
	if len(details.Configs) > 0 {
		fmt.Println("Configuration:")
		fmt.Printf("%-30s %s\n", "KEY", "VALUE")
		fmt.Println(strings.Repeat("-", 80))

		for key, value := range details.Configs {
			fmt.Printf("%-30s %s\n", key, value)
		}
	}

	return nil
}

// displayGroupTable displays consumer groups in table format
func displayGroupTable(groupList *types.GroupList) error {
	if len(groupList.Groups) == 0 {
		fmt.Println("No consumer groups found")
		return nil
	}

	// Print header
	fmt.Printf("%-40s %-15s %-15s %-10s\n", "GROUP ID", "STATE", "PROTOCOL TYPE", "MEMBERS")
	fmt.Println(strings.Repeat("-", 80))

	// Print groups
	for _, group := range groupList.Groups {
		fmt.Printf("%-40s %-15s %-15s %-10d\n",
			group.GroupID, group.State, group.ProtocolType, group.MemberCount)
	}

	// Print pagination info
	if groupList.Pagination != nil {
		fmt.Printf("\nPage %d of %d (%d total groups)\n",
			groupList.Pagination.CurrentPage,
			groupList.Pagination.TotalPages,
			groupList.Pagination.TotalItems)
	}

	return nil
}

// displayGroupDetailsTable displays consumer group details in table format
func displayGroupDetailsTable(details *types.GroupDetails) error {
	fmt.Printf("Consumer Group: %s\n", details.GroupID)
	fmt.Println(strings.Repeat("=", 50))

	// Basic information
	fmt.Printf("State: %s\n", details.State)
	fmt.Printf("Protocol Type: %s\n", details.ProtocolType)
	fmt.Printf("Protocol: %s\n", details.Protocol)
	fmt.Printf("Total Lag: %d\n", details.TotalLag)
	fmt.Println()

	// Coordinator information
	if details.Coordinator != nil {
		fmt.Println("Coordinator:")
		fmt.Printf("  ID: %d\n", details.Coordinator.ID)
		fmt.Printf("  Host: %s\n", details.Coordinator.Host)
		fmt.Printf("  Port: %d\n", details.Coordinator.Port)
		fmt.Println()
	}

	// Member information
	if len(details.Members) > 0 {
		fmt.Println("Members:")
		for i, member := range details.Members {
			fmt.Printf("Member %d:\n", i+1)
			fmt.Printf("  Member ID: %s\n", member.MemberID)
			fmt.Printf("  Client ID: %s\n", member.ClientID)
			fmt.Printf("  Host: %s\n", member.Host)
			fmt.Printf("  Total Lag: %d\n", member.TotalLag)

			if len(member.AssignedPartitions) > 0 {
				fmt.Println("  Assigned Partitions:")
				fmt.Printf("    %-20s %-10s %-15s %-15s %-10s\n", "TOPIC", "PARTITION", "CURRENT OFFSET", "LOG END OFFSET", "LAG")
				fmt.Println("    " + strings.Repeat("-", 70))

				for _, assignment := range member.AssignedPartitions {
					fmt.Printf("    %-20s %-10d %-15d %-15d %-10d\n",
						assignment.Topic,
						assignment.Partition,
						assignment.CurrentOffset,
						assignment.LogEndOffset,
						assignment.Lag)
				}
			}
			fmt.Println()
		}
	}

	return nil
}

// displayMessageTable displays a message in table format
func displayMessageTable(message *types.Message) error {
	fmt.Printf("Topic: %s | Partition: %d | Offset: %d | Timestamp: %s\n",
		message.Topic, message.Partition, message.Offset, message.Timestamp.Format(time.RFC3339))

	if message.Key != "" {
		fmt.Printf("Key: %s\n", message.Key)
	}

	fmt.Printf("Value: %s\n", message.Value)

	if len(message.Headers) > 0 {
		fmt.Println("Headers:")
		for key, value := range message.Headers {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	fmt.Println(strings.Repeat("-", 80))
	return nil
}

// displayProduceResponseTable displays produce response in table format
func displayProduceResponseTable(response *types.ProduceResponse) error {
	fmt.Println("Message produced successfully:")
	fmt.Printf("Topic: %s\n", response.Topic)
	fmt.Printf("Partition: %d\n", response.Partition)
	fmt.Printf("Offset: %d\n", response.Offset)
	fmt.Printf("Timestamp: %s\n", response.Timestamp.Format(time.RFC3339))
	return nil
}

// displayProfileTable displays profiles in table format
func displayProfileTable(profiles []*types.ProfileInfo) error {
	if len(profiles) == 0 {
		fmt.Println("No profiles found")
		return nil
	}

	// Print header
	fmt.Printf("%-20s %-8s %-50s %-8s\n", "NAME", "TYPE", "DETAILS", "ACTIVE")
	fmt.Println(strings.Repeat("-", 86))

	// Print profiles
	for _, profile := range profiles {
		active := ""
		if profile.Active {
			active = "*"
		}
		fmt.Printf("%-20s %-8s %-50s %-8s\n",
			profile.Name, profile.Type, profile.Details, active)
	}

	return nil
}

// formatInt32Slice formats a slice of int32 as a comma-separated string
func formatInt32Slice(slice []int32) string {
	if len(slice) == 0 {
		return "[]"
	}

	strs := make([]string, len(slice))
	for i, v := range slice {
		strs[i] = strconv.Itoa(int(v))
	}

	return "[" + strings.Join(strs, ",") + "]"
}
