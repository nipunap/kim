package cmd

import (
	"context"
	"fmt"
	"strings"

	"kim/internal/client"
	"kim/internal/config"
	"kim/internal/logger"
	"kim/internal/manager"
	"kim/internal/ui"
	"kim/pkg/types"

	"github.com/spf13/cobra"
)

// NewGroupCmd creates the group command
func NewGroupCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage Kafka consumer groups",
		Long:  "Commands for managing Kafka consumer groups including listing, describing, and deleting groups.",
	}

	cmd.AddCommand(NewGroupListCmd(cfg, log))
	cmd.AddCommand(NewGroupDescribeCmd(cfg, log))
	cmd.AddCommand(NewGroupDeleteCmd(cfg, log))
	cmd.AddCommand(NewGroupResetCmd(cfg, log))

	return cmd
}

// NewGroupListCmd creates the group list command
func NewGroupListCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		pattern  string
		page     int
		pageSize int
		sortBy   string
		order    string
		format   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Kafka consumer groups",
		Long:  "List all Kafka consumer groups with optional filtering and pagination.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get active profile
			profile, err := cfg.GetActiveProfile()
			if err != nil {
				return fmt.Errorf("no active profile: %w", err)
			}

			// Create client
			clientManager := client.NewManager(log)
			kafkaClient, err := clientManager.GetClient(profile)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			defer kafkaClient.Close()

			// Create group manager
			groupManager := manager.NewGroupManager(kafkaClient, log)

			// List groups
			opts := &types.ListOptions{
				Page:     page,
				PageSize: pageSize,
				Pattern:  pattern,
				SortBy:   sortBy,
				Order:    order,
			}

			groupList, err := groupManager.ListGroups(context.Background(), opts)
			if err != nil {
				return fmt.Errorf("failed to list consumer groups: %w", err)
			}

			// Display results
			displayOpts := &types.DisplayOptions{
				Format: format,
			}

			return ui.DisplayGroupList(groupList, displayOpts)
		},
	}

	cmd.Flags().StringVar(&pattern, "pattern", "", "filter groups by pattern (supports wildcards)")
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "number of groups per page")
	cmd.Flags().StringVar(&sortBy, "sort-by", "group_id", "sort by field (group_id, state, protocol_type)")
	cmd.Flags().StringVar(&order, "order", "asc", "sort order (asc, desc)")
	cmd.Flags().StringVar(&format, "format", "table", "output format (table, json, yaml)")

	return cmd
}

// NewGroupDescribeCmd creates the group describe command
func NewGroupDescribeCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "describe GROUP_ID",
		Short: "Describe a Kafka consumer group",
		Long:  "Show detailed information about a specific Kafka consumer group including members and lag information.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := args[0]

			// Get active profile
			profile, err := cfg.GetActiveProfile()
			if err != nil {
				return fmt.Errorf("no active profile: %w", err)
			}

			// Create client
			clientManager := client.NewManager(log)
			kafkaClient, err := clientManager.GetClient(profile)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			defer kafkaClient.Close()

			// Create group manager
			groupManager := manager.NewGroupManager(kafkaClient, log)

			// Describe group
			groupDetails, err := groupManager.DescribeGroup(context.Background(), groupID)
			if err != nil {
				return fmt.Errorf("failed to describe consumer group: %w", err)
			}

			// Display results
			displayOpts := &types.DisplayOptions{
				Format: format,
			}

			return ui.DisplayGroupDetails(groupDetails, displayOpts)
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format (table, json, yaml)")

	return cmd
}

// NewGroupDeleteCmd creates the group delete command
func NewGroupDeleteCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete GROUP_ID",
		Short: "Delete a Kafka consumer group",
		Long:  "Delete an existing Kafka consumer group. The group must be empty (no active consumers).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := args[0]

			// Confirm deletion unless force flag is used
			if !force {
				fmt.Printf("Are you sure you want to delete consumer group '%s'? (y/N): ", groupID)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("Consumer group deletion cancelled")
					return nil
				}
			}

			// Get active profile
			profile, err := cfg.GetActiveProfile()
			if err != nil {
				return fmt.Errorf("no active profile: %w", err)
			}

			// Create client
			clientManager := client.NewManager(log)
			kafkaClient, err := clientManager.GetClient(profile)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			defer kafkaClient.Close()

			// Create group manager
			groupManager := manager.NewGroupManager(kafkaClient, log)

			// Delete group
			if err := groupManager.DeleteGroup(context.Background(), groupID); err != nil {
				return fmt.Errorf("failed to delete consumer group: %w", err)
			}

			fmt.Printf("Consumer group '%s' deleted successfully\n", groupID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}

// NewGroupResetCmd creates the group reset command
func NewGroupResetCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		topics     []string
		toEarliest bool
		toLatest   bool
		toOffset   int64
		force      bool
	)

	cmd := &cobra.Command{
		Use:   "reset GROUP_ID",
		Short: "Reset consumer group offsets",
		Long:  "Reset consumer group offsets to earliest, latest, or a specific offset.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := args[0]

			// Validate reset options
			resetOptions := 0
			if toEarliest {
				resetOptions++
			}
			if toLatest {
				resetOptions++
			}
			if cmd.Flags().Changed("to-offset") {
				resetOptions++
			}

			if resetOptions == 0 {
				return fmt.Errorf("must specify one of: --to-earliest, --to-latest, or --to-offset")
			}
			if resetOptions > 1 {
				return fmt.Errorf("can only specify one reset option")
			}

			// Confirm reset unless force flag is used
			if !force {
				fmt.Printf("Are you sure you want to reset offsets for consumer group '%s'? (y/N): ", groupID)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("Offset reset cancelled")
					return nil
				}
			}

			// Get active profile
			profile, err := cfg.GetActiveProfile()
			if err != nil {
				return fmt.Errorf("no active profile: %w", err)
			}

			// Create client
			clientManager := client.NewManager(log)
			kafkaClient, err := clientManager.GetClient(profile)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			defer kafkaClient.Close()

			// Create group manager
			groupManager := manager.NewGroupManager(kafkaClient, log)

			// Build reset request
			req := &types.ResetOffsetsRequest{
				GroupID:    groupID,
				Topics:     topics,
				ToEarliest: toEarliest,
				ToLatest:   toLatest,
			}

			if cmd.Flags().Changed("to-offset") {
				req.ToOffset = &toOffset
			}

			// Reset offsets
			if err := groupManager.ResetGroupOffsets(context.Background(), req); err != nil {
				return fmt.Errorf("failed to reset consumer group offsets: %w", err)
			}

			fmt.Printf("Consumer group '%s' offsets reset successfully\n", groupID)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&topics, "topics", nil, "topics to reset (default: all topics)")
	cmd.Flags().BoolVar(&toEarliest, "to-earliest", false, "reset to earliest offset")
	cmd.Flags().BoolVar(&toLatest, "to-latest", false, "reset to latest offset")
	cmd.Flags().Int64Var(&toOffset, "to-offset", 0, "reset to specific offset")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}
