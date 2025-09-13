package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nipunap/kim/internal/client"
	"github.com/nipunap/kim/internal/config"
	"github.com/nipunap/kim/internal/logger"
	"github.com/nipunap/kim/internal/manager"
	"github.com/nipunap/kim/internal/ui"
	"github.com/nipunap/kim/pkg/types"

	"github.com/spf13/cobra"
)

// NewMessageCmd creates the message command
func NewMessageCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "message",
		Short: "Manage Kafka messages",
		Long:  "Commands for producing and consuming Kafka messages.",
	}

	cmd.AddCommand(NewMessageProduceCmd(cfg, log))
	cmd.AddCommand(NewMessageConsumeCmd(cfg, log))

	return cmd
}

// NewMessageProduceCmd creates the message produce command
func NewMessageProduceCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		key       string
		value     string
		partition int32
		headers   []string
		format    string
	)

	cmd := &cobra.Command{
		Use:   "produce TOPIC",
		Short: "Produce a message to a Kafka topic",
		Long:  "Produce a message to a Kafka topic with optional key, partition, and headers.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := args[0]

			if value == "" {
				return fmt.Errorf("message value is required (use --value flag)")
			}

			// Parse headers
			headerMap := make(map[string]string)
			for _, header := range headers {
				parts := strings.SplitN(header, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header format: %s (expected key=value)", header)
				}
				headerMap[parts[0]] = parts[1]
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

			// Create message manager
			messageManager := manager.NewMessageManager(kafkaClient, log)

			// Build produce request
			req := &types.ProduceRequest{
				Topic:   topic,
				Key:     key,
				Value:   value,
				Headers: headerMap,
			}

			if cmd.Flags().Changed("partition") {
				req.Partition = &partition
			}

			// Produce message
			response, err := messageManager.ProduceMessage(context.Background(), req)
			if err != nil {
				return fmt.Errorf("failed to produce message: %w", err)
			}

			// Display result
			displayOpts := &types.DisplayOptions{
				Format: format,
			}

			return ui.DisplayProduceResponse(response, displayOpts)
		},
	}

	cmd.Flags().StringVar(&key, "key", "", "message key")
	cmd.Flags().StringVar(&value, "value", "", "message value (required)")
	cmd.Flags().Int32Var(&partition, "partition", -1, "specific partition to produce to")
	cmd.Flags().StringSliceVar(&headers, "header", nil, "message headers (key=value)")
	cmd.Flags().StringVar(&format, "format", "table", "output format (table, json, yaml)")

	cmd.MarkFlagRequired("value")

	return cmd
}

// NewMessageConsumeCmd creates the message consume command
func NewMessageConsumeCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		groupID       string
		partition     int32
		fromBeginning bool
		maxMessages   int
		timeout       time.Duration
		format        string
	)

	cmd := &cobra.Command{
		Use:   "consume TOPIC",
		Short: "Consume messages from a Kafka topic",
		Long:  "Consume messages from a Kafka topic with real-time streaming or batch processing.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := args[0]

			if groupID == "" {
				return fmt.Errorf("consumer group ID is required (use --group-id flag)")
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

			// Create message manager
			messageManager := manager.NewMessageManager(kafkaClient, log)

			// Build consume request
			req := &types.ConsumeRequest{
				Topic:         topic,
				Partition:     partition,
				GroupID:       groupID,
				FromBeginning: fromBeginning,
			}

			// Start consumer
			messages, errors, err := messageManager.StartConsumer(context.Background(), req)
			if err != nil {
				return fmt.Errorf("failed to start consumer: %w", err)
			}

			// Setup signal handling for graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			// Setup timeout if specified
			var timeoutChan <-chan time.Time
			if timeout > 0 {
				timeoutChan = time.After(timeout)
			}

			fmt.Printf("Started consuming from topic '%s' (partition %d, group '%s')\n", topic, partition, groupID)
			fmt.Println("Press Ctrl+C to stop consuming...")

			messageCount := 0
			displayOpts := &types.DisplayOptions{
				Format: format,
			}

			// Consume messages
			for {
				select {
				case message := <-messages:
					if message == nil {
						fmt.Println("Consumer closed")
						return nil
					}

					if err := ui.DisplayMessage(message, displayOpts); err != nil {
						log.Error("Failed to display message", "error", err)
					}

					messageCount++
					if maxMessages > 0 && messageCount >= maxMessages {
						fmt.Printf("Reached maximum message count (%d), stopping consumer\n", maxMessages)
						return messageManager.StopConsumer(topic, groupID, partition)
					}

				case err := <-errors:
					if err != nil {
						log.Error("Consumer error", "error", err)
					}

				case <-sigChan:
					fmt.Println("\nReceived interrupt signal, stopping consumer...")
					return messageManager.StopConsumer(topic, groupID, partition)

				case <-timeoutChan:
					fmt.Printf("Timeout reached (%v), stopping consumer\n", timeout)
					return messageManager.StopConsumer(topic, groupID, partition)
				}
			}
		},
	}

	cmd.Flags().StringVar(&groupID, "group-id", "", "consumer group ID (required)")
	cmd.Flags().Int32Var(&partition, "partition", 0, "partition to consume from")
	cmd.Flags().BoolVar(&fromBeginning, "from-beginning", false, "consume from the beginning of the topic")
	cmd.Flags().IntVar(&maxMessages, "max-messages", 0, "maximum number of messages to consume (0 = unlimited)")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "timeout for consuming messages (0 = no timeout)")
	cmd.Flags().StringVar(&format, "format", "table", "output format (table, json, yaml)")

	cmd.MarkFlagRequired("group-id")

	return cmd
}
