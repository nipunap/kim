package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/nipunap/kim/internal/config"
)

const (
	kafkaBootstrapServers = "localhost:9092"
	kafkaSASLServers      = "localhost:9093"
	testTimeout           = 30 * time.Second
	dockerComposeFile     = "docker-compose.test.yml"
)

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Check if Docker is available
	if !isDockerAvailable() {
		fmt.Println("Docker not available, skipping integration tests")
		os.Exit(0)
	}

	// Start Kafka cluster
	if err := startKafkaCluster(); err != nil {
		fmt.Printf("Failed to start Kafka cluster: %v\n", err)
		os.Exit(1)
	}

	// Wait for Kafka to be ready
	if err := waitForKafka(); err != nil {
		fmt.Printf("Kafka cluster not ready: %v\n", err)
		stopKafkaCluster()
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	stopKafkaCluster()
	os.Exit(code)
}

func TestIntegrationProfileManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary config directory
	tempDir, err := os.MkdirTemp("", "kim-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Test profile add
	output, err := runKimCommand("profile", "add", "test-integration",
		"--type", "kafka",
		"--bootstrap-servers", kafkaBootstrapServers,
		"--security-protocol", "PLAINTEXT")
	if err != nil {
		t.Fatalf("Failed to add profile: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Profile 'test-integration' added successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Test profile list
	output, err = runKimCommand("profile", "list")
	if err != nil {
		t.Fatalf("Failed to list profiles: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "test-integration") {
		t.Errorf("Profile should be listed, got: %s", output)
	}

	// Test profile use
	output, err = runKimCommand("profile", "use", "test-integration")
	if err != nil {
		t.Fatalf("Failed to use profile: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Active profile set to 'test-integration'") {
		t.Errorf("Expected profile activation message, got: %s", output)
	}
}

func TestIntegrationTopicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupTestProfile(t)

	testTopicName := "kim-test-topic-" + fmt.Sprintf("%d", time.Now().Unix())

	// Test topic creation
	output, err := runKimCommand("topic", "create", testTopicName,
		"--partitions", "3",
		"--replication-factor", "1",
		"--config", "retention.ms=86400000")
	if err != nil {
		t.Fatalf("Failed to create topic: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, fmt.Sprintf("Topic '%s' created successfully", testTopicName)) {
		t.Errorf("Expected topic creation success, got: %s", output)
	}

	// Test topic list
	output, err = runKimCommand("topic", "list")
	if err != nil {
		t.Fatalf("Failed to list topics: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, testTopicName) {
		t.Errorf("Topic should be listed, got: %s", output)
	}

	// Test topic describe
	output, err = runKimCommand("topic", "describe", testTopicName)
	if err != nil {
		t.Fatalf("Failed to describe topic: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, testTopicName) {
		t.Errorf("Topic description should contain topic name, got: %s", output)
	}
	if !strings.Contains(output, "Partitions: 3") {
		t.Errorf("Topic should have 3 partitions, got: %s", output)
	}

	// Test topic list with pagination
	output, err = runKimCommand("topic", "list", "--page-size", "5", "--page", "1")
	if err != nil {
		t.Fatalf("Failed to list topics with pagination: %v\nOutput: %s", err, output)
	}

	// Test topic list with filter
	output, err = runKimCommand("topic", "list", "--filter", testTopicName)
	if err != nil {
		t.Fatalf("Failed to list topics with filter: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, testTopicName) {
		t.Errorf("Filtered topic list should contain test topic, got: %s", output)
	}

	// Test topic delete
	output, err = runKimCommand("topic", "delete", testTopicName, "--confirm")
	if err != nil {
		t.Fatalf("Failed to delete topic: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, fmt.Sprintf("Topic '%s' deleted successfully", testTopicName)) {
		t.Errorf("Expected topic deletion success, got: %s", output)
	}

	// Verify topic is deleted
	time.Sleep(2 * time.Second) // Wait for deletion to propagate
	output, err = runKimCommand("topic", "list")
	if err != nil {
		t.Fatalf("Failed to list topics after deletion: %v\nOutput: %s", err, output)
	}

	if strings.Contains(output, testTopicName) {
		t.Errorf("Topic should be deleted, but still found in: %s", output)
	}
}

func TestIntegrationMessageOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupTestProfile(t)

	testTopicName := "kim-test-messages-" + fmt.Sprintf("%d", time.Now().Unix())

	// Create topic for message testing
	_, err := runKimCommand("topic", "create", testTopicName,
		"--partitions", "1",
		"--replication-factor", "1")
	if err != nil {
		t.Fatalf("Failed to create topic for message testing: %v", err)
	}
	defer runKimCommand("topic", "delete", testTopicName, "--confirm")

	// Test message production
	testKey := "test-key-123"
	testValue := "test-value-hello-world"
	output, err := runKimCommand("message", "produce", testTopicName,
		"--key", testKey,
		"--value", testValue,
		"--header", "test-header:test-header-value")
	if err != nil {
		t.Fatalf("Failed to produce message: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Message produced successfully") {
		t.Errorf("Expected message production success, got: %s", output)
	}

	// Test message consumption
	output, err = runKimCommandWithTimeout("message", "consume", testTopicName,
		"--group", "kim-test-group",
		"--from-beginning",
		"--max-messages", "1")
	if err != nil {
		t.Fatalf("Failed to consume message: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, testKey) {
		t.Errorf("Consumed message should contain key, got: %s", output)
	}
	if !strings.Contains(output, testValue) {
		t.Errorf("Consumed message should contain value, got: %s", output)
	}
	if !strings.Contains(output, "test-header") {
		t.Errorf("Consumed message should contain header, got: %s", output)
	}

	// Test batch message production
	for i := 0; i < 5; i++ {
		_, err := runKimCommand("message", "produce", testTopicName,
			"--key", fmt.Sprintf("batch-key-%d", i),
			"--value", fmt.Sprintf("batch-value-%d", i))
		if err != nil {
			t.Fatalf("Failed to produce batch message %d: %v", i, err)
		}
	}

	// Test consuming multiple messages
	output, err = runKimCommandWithTimeout("message", "consume", testTopicName,
		"--group", "kim-test-batch-group",
		"--from-beginning",
		"--max-messages", "5")
	if err != nil {
		t.Fatalf("Failed to consume batch messages: %v\nOutput: %s", err, output)
	}

	for i := 0; i < 5; i++ {
		expectedKey := fmt.Sprintf("batch-key-%d", i)
		if !strings.Contains(output, expectedKey) {
			t.Errorf("Batch consumption should contain %s, got: %s", expectedKey, output)
		}
	}
}

func TestIntegrationConsumerGroupOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupTestProfile(t)

	testTopicName := "kim-test-groups-" + fmt.Sprintf("%d", time.Now().Unix())
	testGroupName := "kim-test-consumer-group-" + fmt.Sprintf("%d", time.Now().Unix())

	// Create topic
	_, err := runKimCommand("topic", "create", testTopicName,
		"--partitions", "2",
		"--replication-factor", "1")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}
	defer runKimCommand("topic", "delete", testTopicName, "--confirm")

	// Produce some messages
	for i := 0; i < 10; i++ {
		_, err := runKimCommand("message", "produce", testTopicName,
			"--key", fmt.Sprintf("group-test-key-%d", i),
			"--value", fmt.Sprintf("group-test-value-%d", i))
		if err != nil {
			t.Fatalf("Failed to produce message for group testing: %v", err)
		}
	}

	// Start a consumer to create the group
	go func() {
		runKimCommandWithTimeout("message", "consume", testTopicName,
			"--group", testGroupName,
			"--from-beginning",
			"--max-messages", "5")
	}()

	// Wait for consumer group to be created
	time.Sleep(3 * time.Second)

	// Test group list
	output, err := runKimCommand("group", "list")
	if err != nil {
		t.Fatalf("Failed to list consumer groups: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, testGroupName) {
		t.Errorf("Consumer group should be listed, got: %s", output)
	}

	// Test group describe
	output, err = runKimCommand("group", "describe", testGroupName)
	if err != nil {
		t.Fatalf("Failed to describe consumer group: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, testGroupName) {
		t.Errorf("Group description should contain group name, got: %s", output)
	}

	// Test group list with filter
	output, err = runKimCommand("group", "list", "--filter", testGroupName)
	if err != nil {
		t.Fatalf("Failed to list groups with filter: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, testGroupName) {
		t.Errorf("Filtered group list should contain test group, got: %s", output)
	}

	// Wait for consumer to finish
	time.Sleep(2 * time.Second)

	// Test group reset offsets
	output, err = runKimCommand("group", "reset", testGroupName,
		"--topic", testTopicName,
		"--to", "earliest")
	if err != nil {
		t.Fatalf("Failed to reset group offsets: %v\nOutput: %s", err, output)
	}

	// Test group delete
	output, err = runKimCommand("group", "delete", testGroupName, "--confirm")
	if err != nil {
		t.Fatalf("Failed to delete consumer group: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, fmt.Sprintf("Consumer group '%s' deleted successfully", testGroupName)) {
		t.Errorf("Expected group deletion success, got: %s", output)
	}
}

func TestIntegrationOutputFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupTestProfile(t)

	// Test JSON output format
	output, err := runKimCommand("topic", "list", "--format", "json")
	if err != nil {
		t.Fatalf("Failed to list topics in JSON format: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "{") && !strings.Contains(output, "[") {
		t.Errorf("JSON output should contain JSON structure, got: %s", output)
	}

	// Test YAML output format
	output, err = runKimCommand("topic", "list", "--format", "yaml")
	if err != nil {
		t.Fatalf("Failed to list topics in YAML format: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, ":") && !strings.Contains(output, "-") {
		t.Errorf("YAML output should contain YAML structure, got: %s", output)
	}

	// Test table output format (default)
	output, err = runKimCommand("topic", "list", "--format", "table")
	if err != nil {
		t.Fatalf("Failed to list topics in table format: %v\nOutput: %s", err, output)
	}

	// Table format should have headers
	if !strings.Contains(output, "NAME") {
		t.Errorf("Table output should contain headers, got: %s", output)
	}
}

func TestIntegrationErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupTestProfile(t)

	// Test creating topic with invalid parameters
	_, err := runKimCommand("topic", "create", "invalid-topic",
		"--partitions", "0") // Invalid partition count
	if err == nil {
		t.Error("Should fail with invalid partition count")
	}

	// Test describing non-existent topic
	_, err = runKimCommand("topic", "describe", "non-existent-topic-12345")
	if err == nil {
		t.Error("Should fail when describing non-existent topic")
	}

	// Test deleting non-existent topic
	_, err = runKimCommand("topic", "delete", "non-existent-topic-12345", "--confirm")
	if err == nil {
		t.Error("Should fail when deleting non-existent topic")
	}

	// Test describing non-existent consumer group
	_, err = runKimCommand("group", "describe", "non-existent-group-12345")
	if err == nil {
		t.Error("Should fail when describing non-existent group")
	}

	// Test producing to non-existent topic (should auto-create)
	nonExistentTopic := "auto-created-topic-" + fmt.Sprintf("%d", time.Now().Unix())
	output, err := runKimCommand("message", "produce", nonExistentTopic,
		"--value", "test-auto-create")
	if err != nil {
		t.Fatalf("Should auto-create topic when producing: %v\nOutput: %s", err, output)
	}

	// Clean up auto-created topic
	defer runKimCommand("topic", "delete", nonExistentTopic, "--confirm")
}

// Helper functions

func setupTestProfile(t *testing.T) {
	// Create temporary config directory
	tempDir, err := os.MkdirTemp("", "kim-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	// Add test profile
	_, err = runKimCommand("profile", "add", "integration-test",
		"--type", "kafka",
		"--bootstrap-servers", kafkaBootstrapServers,
		"--security-protocol", "PLAINTEXT")
	if err != nil {
		t.Fatalf("Failed to add test profile: %v", err)
	}

	// Set as active profile
	_, err = runKimCommand("profile", "use", "integration-test")
	if err != nil {
		t.Fatalf("Failed to set active profile: %v", err)
	}
}

func runKimCommand(args ...string) (string, error) {
	return runKimCommandWithTimeout(args...)
}

func runKimCommandWithTimeout(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Build kim binary if it doesn't exist
	kimBinary := "./kim"
	if _, err := os.Stat(kimBinary); os.IsNotExist(err) {
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", kimBinary, "./cmd/kim")
		if err := buildCmd.Run(); err != nil {
			return "", fmt.Errorf("failed to build kim binary: %w", err)
		}
	}

	cmd := exec.CommandContext(ctx, kimBinary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

func startKafkaCluster() error {
	cmd := exec.Command("docker-compose", "-f", dockerComposeFile, "up", "-d", "--wait")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopKafkaCluster() error {
	cmd := exec.Command("docker-compose", "-f", dockerComposeFile, "down", "-v")
	return cmd.Run()
}

func waitForKafka() error {
	// Wait for Kafka to be ready by trying to connect
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		// Try to create a temporary config and test connection
		tempDir, err := os.MkdirTemp("", "kim-wait-*")
		if err != nil {
			continue
		}
		defer os.RemoveAll(tempDir)

		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)

		// Create config
		cfg, err := config.New()
		if err != nil {
			os.Setenv("HOME", oldHome)
			continue
		}

		// Add test profile
		profile := &config.Profile{
			Name:             "wait-test",
			Type:             "kafka",
			BootstrapServers: kafkaBootstrapServers,
			SecurityProtocol: "PLAINTEXT",
		}

		err = cfg.AddProfile(profile)
		if err != nil {
			os.Setenv("HOME", oldHome)
			continue
		}

		err = cfg.SetActiveProfile("wait-test")
		if err != nil {
			os.Setenv("HOME", oldHome)
			continue
		}

		// Try to list topics (this will test the connection)
		output, err := runKimCommand("topic", "list")
		os.Setenv("HOME", oldHome)

		if err == nil || strings.Contains(output, "NAME") {
			fmt.Println("Kafka cluster is ready!")
			return nil
		}

		fmt.Printf("Waiting for Kafka... (attempt %d/%d)\n", i+1, maxRetries)
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("kafka cluster not ready after %d attempts", maxRetries)
}
