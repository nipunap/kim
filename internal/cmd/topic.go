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

// NewTopicCmd creates the topic command
func NewTopicCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topic",
		Short: "Manage Kafka topics",
		Long:  "Commands for managing Kafka topics including listing, describing, creating, and deleting topics.",
	}

	cmd.AddCommand(NewTopicListCmd(cfg, log))
	cmd.AddCommand(NewTopicDescribeCmd(cfg, log))
	cmd.AddCommand(NewTopicCreateCmd(cfg, log))
	cmd.AddCommand(NewTopicDeleteCmd(cfg, log))

	return cmd
}

// NewTopicListCmd creates the topic list command
func NewTopicListCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
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
		Short: "List Kafka topics",
		Long:  "List all Kafka topics with optional filtering and pagination.",
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

			// Create topic manager
			topicManager := manager.NewTopicManager(kafkaClient, log)

			// List topics
			opts := &types.ListOptions{
				Page:     page,
				PageSize: pageSize,
				Pattern:  pattern,
				SortBy:   sortBy,
				Order:    order,
			}

			topicList, err := topicManager.ListTopics(context.Background(), opts)
			if err != nil {
				return fmt.Errorf("failed to list topics: %w", err)
			}

			// Display results
			displayOpts := &types.DisplayOptions{
				Format: format,
			}

			return ui.DisplayTopicList(topicList, displayOpts)
		},
	}

	cmd.Flags().StringVar(&pattern, "pattern", "", "filter topics by pattern (supports wildcards)")
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "number of topics per page")
	cmd.Flags().StringVar(&sortBy, "sort-by", "name", "sort by field (name, partitions, replication_factor)")
	cmd.Flags().StringVar(&order, "order", "asc", "sort order (asc, desc)")
	cmd.Flags().StringVar(&format, "format", "table", "output format (table, json, yaml)")

	return cmd
}

// NewTopicDescribeCmd creates the topic describe command
func NewTopicDescribeCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "describe TOPIC_NAME",
		Short: "Describe a Kafka topic",
		Long:  "Show detailed information about a specific Kafka topic including configuration and partition details.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topicName := args[0]

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

			// Create topic manager
			topicManager := manager.NewTopicManager(kafkaClient, log)

			// Describe topic
			topicDetails, err := topicManager.DescribeTopic(context.Background(), topicName)
			if err != nil {
				return fmt.Errorf("failed to describe topic: %w", err)
			}

			// Display results
			displayOpts := &types.DisplayOptions{
				Format: format,
			}

			return ui.DisplayTopicDetails(topicDetails, displayOpts)
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format (table, json, yaml)")

	return cmd
}

// NewTopicCreateCmd creates the topic create command
func NewTopicCreateCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		partitions        int32
		replicationFactor int16
		configs           []string
	)

	cmd := &cobra.Command{
		Use:   "create TOPIC_NAME",
		Short: "Create a Kafka topic",
		Long:  "Create a new Kafka topic with specified configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topicName := args[0]

			// Parse config entries
			configMap := make(map[string]string)
			for _, config := range configs {
				parts := strings.SplitN(config, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid config format: %s (expected key=value)", config)
				}
				configMap[parts[0]] = parts[1]
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

			// Create topic manager
			topicManager := manager.NewTopicManager(kafkaClient, log)

			// Create topic
			req := &types.CreateTopicRequest{
				Name:              topicName,
				Partitions:        partitions,
				ReplicationFactor: replicationFactor,
				Configs:           configMap,
			}

			if err := topicManager.CreateTopic(context.Background(), req); err != nil {
				return fmt.Errorf("failed to create topic: %w", err)
			}

			fmt.Printf("Topic '%s' created successfully\n", topicName)
			return nil
		},
	}

	cmd.Flags().Int32Var(&partitions, "partitions", 1, "number of partitions")
	cmd.Flags().Int16Var(&replicationFactor, "replication-factor", 1, "replication factor")
	cmd.Flags().StringSliceVar(&configs, "config", nil, "topic configuration (key=value)")

	return cmd
}

// NewTopicDeleteCmd creates the topic delete command
func NewTopicDeleteCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete TOPIC_NAME",
		Short: "Delete a Kafka topic",
		Long:  "Delete an existing Kafka topic. This operation is irreversible.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topicName := args[0]

			// Confirm deletion unless force flag is used
			if !force {
				fmt.Printf("Are you sure you want to delete topic '%s'? This operation is irreversible. (y/N): ", topicName)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("Topic deletion cancelled")
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

			// Create topic manager
			topicManager := manager.NewTopicManager(kafkaClient, log)

			// Delete topic
			if err := topicManager.DeleteTopic(context.Background(), topicName); err != nil {
				return fmt.Errorf("failed to delete topic: %w", err)
			}

			fmt.Printf("Topic '%s' deleted successfully\n", topicName)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}
