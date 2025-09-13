package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kim/internal/testutil"

	"github.com/spf13/cobra"
)

// setupTestEnvironment creates a temporary test environment
func setupTestEnvironment(t *testing.T) (string, func()) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "kim-integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	// Cleanup function
	cleanup := func() {
		os.Setenv("HOME", oldHome)
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// executeCommand executes a cobra command and returns output and error
func executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return buf.String(), err
}

func TestRootCommand(t *testing.T) {
	tempDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create test config
	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	// Create root command
	rootCmd := NewRootCmd(cfg, log)

	// Test help command
	output, err := executeCommand(rootCmd, "--help")
	if err != nil {
		t.Errorf("Help command failed: %v", err)
	}

	if !strings.Contains(output, "Kim - Kafka Interactive Manager") {
		t.Error("Help output should contain application description")
	}

	// Test version information
	if !strings.Contains(output, "Usage:") {
		t.Error("Help output should contain usage information")
	}

	// Verify config file was created
	configPath := filepath.Join(tempDir, ".kim", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file should be created at %s", configPath)
	}
}

func TestProfileCommands(t *testing.T) {
	tempDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	// Test profile list command
	profileCmd := NewProfileCmd(cfg, log)
	output, err := executeCommand(profileCmd, "list")
	if err != nil {
		t.Errorf("Profile list command failed: %v", err)
	}

	if !strings.Contains(output, "test-kafka") {
		t.Error("Profile list should contain test profiles")
	}

	// Test profile add command
	output, err = executeCommand(profileCmd, "add", "test-new",
		"--type", "kafka",
		"--bootstrap-servers", "localhost:9093",
		"--security-protocol", "PLAINTEXT")
	if err != nil {
		t.Errorf("Profile add command failed: %v", err)
	}

	if !strings.Contains(output, "Profile 'test-new' added successfully") {
		t.Error("Profile add should show success message")
	}

	// Verify profile was added
	output, err = executeCommand(profileCmd, "list")
	if err != nil {
		t.Errorf("Profile list command failed: %v", err)
	}

	if !strings.Contains(output, "test-new") {
		t.Error("Profile list should contain newly added profile")
	}

	// Test profile use command
	output, err = executeCommand(profileCmd, "use", "test-new")
	if err != nil {
		t.Errorf("Profile use command failed: %v", err)
	}

	if !strings.Contains(output, "Active profile set to 'test-new'") {
		t.Error("Profile use should show success message")
	}

	// Test profile delete command
	output, err = executeCommand(profileCmd, "delete", "test-new")
	if err != nil {
		t.Errorf("Profile delete command failed: %v", err)
	}

	if !strings.Contains(output, "Profile 'test-new' deleted successfully") {
		t.Error("Profile delete should show success message")
	}
}

func TestProfileAddMSK(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	profileCmd := NewProfileCmd(cfg, log)

	// Test MSK profile add
	output, err := executeCommand(profileCmd, "add", "test-msk-new",
		"--type", "msk",
		"--region", "us-west-2",
		"--cluster-arn", "arn:aws:kafka:us-west-2:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1",
		"--auth-method", "IAM")
	if err != nil {
		t.Errorf("MSK profile add command failed: %v", err)
	}

	if !strings.Contains(output, "Profile 'test-msk-new' added successfully") {
		t.Error("MSK profile add should show success message")
	}

	// Verify MSK profile was added
	output, err = executeCommand(profileCmd, "list")
	if err != nil {
		t.Errorf("Profile list command failed: %v", err)
	}

	if !strings.Contains(output, "test-msk-new") {
		t.Error("Profile list should contain newly added MSK profile")
	}
	if !strings.Contains(output, "msk") {
		t.Error("Profile list should show MSK type")
	}
}

func TestProfileAddSSL(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	profileCmd := NewProfileCmd(cfg, log)

	// Test SSL profile add
	output, err := executeCommand(profileCmd, "add", "test-ssl",
		"--type", "kafka",
		"--bootstrap-servers", "localhost:9093",
		"--security-protocol", "SSL",
		"--ssl-ca-file", "/path/to/ca.pem",
		"--ssl-cert-file", "/path/to/cert.pem",
		"--ssl-key-file", "/path/to/key.pem")
	if err != nil {
		t.Errorf("SSL profile add command failed: %v", err)
	}

	if !strings.Contains(output, "Profile 'test-ssl' added successfully") {
		t.Error("SSL profile add should show success message")
	}
}

func TestProfileAddSASL(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	profileCmd := NewProfileCmd(cfg, log)

	// Test SASL profile add
	output, err := executeCommand(profileCmd, "add", "test-sasl",
		"--type", "kafka",
		"--bootstrap-servers", "localhost:9093",
		"--security-protocol", "SASL_PLAINTEXT",
		"--sasl-mechanism", "PLAIN",
		"--sasl-username", "testuser",
		"--sasl-password", "testpass")
	if err != nil {
		t.Errorf("SASL profile add command failed: %v", err)
	}

	if !strings.Contains(output, "Profile 'test-sasl' added successfully") {
		t.Error("SASL profile add should show success message")
	}
}

func TestTopicCommands(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	// Create topic command (these will fail without real Kafka, but we test the CLI structure)
	topicCmd := NewTopicCmd(cfg, log)

	// Test topic list command structure
	_, err := executeCommand(topicCmd, "list", "--help")
	if err != nil {
		t.Errorf("Topic list help failed: %v", err)
	}

	// Test topic create command structure
	_, err = executeCommand(topicCmd, "create", "--help")
	if err != nil {
		t.Errorf("Topic create help failed: %v", err)
	}

	// Test topic describe command structure
	_, err = executeCommand(topicCmd, "describe", "--help")
	if err != nil {
		t.Errorf("Topic describe help failed: %v", err)
	}

	// Test topic delete command structure
	_, err = executeCommand(topicCmd, "delete", "--help")
	if err != nil {
		t.Errorf("Topic delete help failed: %v", err)
	}
}

func TestGroupCommands(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	// Create group command
	groupCmd := NewGroupCmd(cfg, log)

	// Test group list command structure
	_, err := executeCommand(groupCmd, "list", "--help")
	if err != nil {
		t.Errorf("Group list help failed: %v", err)
	}

	// Test group describe command structure
	_, err = executeCommand(groupCmd, "describe", "--help")
	if err != nil {
		t.Errorf("Group describe help failed: %v", err)
	}

	// Test group delete command structure
	_, err = executeCommand(groupCmd, "delete", "--help")
	if err != nil {
		t.Errorf("Group delete help failed: %v", err)
	}

	// Test group reset command structure
	_, err = executeCommand(groupCmd, "reset", "--help")
	if err != nil {
		t.Errorf("Group reset help failed: %v", err)
	}
}

func TestMessageCommands(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	// Create message command
	messageCmd := NewMessageCmd(cfg, log)

	// Test message consume command structure
	_, err := executeCommand(messageCmd, "consume", "--help")
	if err != nil {
		t.Errorf("Message consume help failed: %v", err)
	}

	// Test message produce command structure
	_, err = executeCommand(messageCmd, "produce", "--help")
	if err != nil {
		t.Errorf("Message produce help failed: %v", err)
	}
}

func TestCommandFlags(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	// Test global flags
	rootCmd := NewRootCmd(cfg, log)

	// Test debug flag
	output, err := executeCommand(rootCmd, "--debug", "--help")
	if err != nil {
		t.Errorf("Debug flag failed: %v", err)
	}

	if !strings.Contains(output, "Kim - Kafka Interactive Manager") {
		t.Error("Debug flag should not affect help output")
	}

	// Test config flag
	tempConfig := filepath.Join(os.TempDir(), "test-config.yaml")
	output, err = executeCommand(rootCmd, "--config", tempConfig, "--help")
	if err != nil {
		t.Errorf("Config flag failed: %v", err)
	}

	if !strings.Contains(output, "Usage:") {
		t.Error("Config flag should not affect help output")
	}
}

func TestInvalidCommands(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	rootCmd := NewRootCmd(cfg, log)

	// Test invalid subcommand
	_, err := executeCommand(rootCmd, "invalid-command")
	if err == nil {
		t.Error("Invalid command should return error")
	}

	// Test profile with invalid subcommand
	profileCmd := NewProfileCmd(cfg, log)
	_, err = executeCommand(profileCmd, "invalid-subcommand")
	if err == nil {
		t.Error("Invalid profile subcommand should return error")
	}
}

func TestProfileValidation(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	profileCmd := NewProfileCmd(cfg, log)

	// Test profile add without required fields
	_, err := executeCommand(profileCmd, "add", "invalid-profile")
	if err == nil {
		t.Error("Profile add without type should fail")
	}

	// Test profile add with invalid type
	_, err = executeCommand(profileCmd, "add", "invalid-profile", "--type", "invalid")
	if err == nil {
		t.Error("Profile add with invalid type should fail")
	}

	// Test MSK profile without required fields
	_, err = executeCommand(profileCmd, "add", "invalid-msk",
		"--type", "msk")
	if err == nil {
		t.Error("MSK profile without region should fail")
	}

	// Test Kafka profile without bootstrap servers
	_, err = executeCommand(profileCmd, "add", "invalid-kafka",
		"--type", "kafka")
	if err == nil {
		t.Error("Kafka profile without bootstrap servers should fail")
	}
}

func TestOutputFormats(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	profileCmd := NewProfileCmd(cfg, log)

	// Test JSON output format
	output, err := executeCommand(profileCmd, "list", "--format", "json")
	if err != nil {
		t.Errorf("JSON format failed: %v", err)
	}

	// Should contain JSON structure (even if empty)
	if !strings.Contains(output, "{") && !strings.Contains(output, "[") {
		t.Error("JSON format should produce JSON output")
	}

	// Test YAML output format
	output, err = executeCommand(profileCmd, "list", "--format", "yaml")
	if err != nil {
		t.Errorf("YAML format failed: %v", err)
	}

	// Should contain YAML structure
	if !strings.Contains(output, ":") && !strings.Contains(output, "-") {
		t.Error("YAML format should produce YAML output")
	}

	// Test invalid output format
	_, err = executeCommand(profileCmd, "list", "--format", "invalid")
	if err == nil {
		t.Error("Invalid output format should fail")
	}
}
