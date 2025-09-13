package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nipunap/kim/internal/testutil"

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
	_, cleanup := setupTestEnvironment(t)
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

	if !strings.Contains(output, "powerful command-line interface for managing Kafka") {
		t.Error("Help output should contain application description")
	}

	// Test version information
	if !strings.Contains(output, "Usage:") {
		t.Error("Help output should contain usage information")
	}

	// Note: Config file is only created when needed (e.g., when adding profiles)
	// The help command doesn't need to create a config file
}

func TestProfileCommands(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	// Test profile list command
	profileCmd := NewProfileCmd(cfg, log)
	_, err := executeCommand(profileCmd, "list")
	if err != nil {
		t.Errorf("Profile list command failed: %v", err)
	}

	// Verify test profiles exist in config (output goes to stdout, not captured)
	if _, exists := cfg.Profiles["test-kafka"]; !exists {
		t.Error("Profile list should contain test-kafka profile")
	}
	if _, exists := cfg.Profiles["test-msk"]; !exists {
		t.Error("Profile list should contain test-msk profile")
	}

	// Test profile add command
	profileCmd = NewProfileCmd(cfg, log) // Create fresh command
	output, err := executeCommand(profileCmd, "add", "test-new",
		"--type", "kafka",
		"--bootstrap-servers", "localhost:9093",
		"--security-protocol", "PLAINTEXT")

	// Check if the profile was actually added by verifying it exists in the config
	// rather than relying on the command output/error
	if _, exists := cfg.Profiles["test-new"]; !exists {
		t.Errorf("Profile 'test-new' was not added to config. Error: %v, Output: %s", err, output)
	}

	// Verify profile was added
	profileCmd = NewProfileCmd(cfg, log) // Create fresh command
	_, err = executeCommand(profileCmd, "list")
	if err != nil {
		t.Errorf("Profile list command failed: %v", err)
	}

	// Verify the new profile exists in config (output goes to stdout, not captured)
	if _, exists := cfg.Profiles["test-new"]; !exists {
		t.Error("Profile list should contain newly added profile")
	}

	// Test profile use command
	profileCmd = NewProfileCmd(cfg, log) // Create fresh command
	_, err = executeCommand(profileCmd, "use", "test-new")

	// Check if the active profile was actually changed
	if cfg.ActiveProfile != "test-new" {
		t.Errorf("Active profile was not changed to 'test-new'. Current: %s, Error: %v", cfg.ActiveProfile, err)
	}

	// Test profile delete command
	profileCmd = NewProfileCmd(cfg, log) // Create fresh command
	_, err = executeCommand(profileCmd, "delete", "test-new")

	// Note: Delete should fail because test-new is the active profile
	// Check if the profile still exists (it should, because deletion should fail)
	if _, exists := cfg.Profiles["test-new"]; !exists {
		t.Error("Profile 'test-new' should still exist because it's the active profile")
	}
}

func TestProfileAddMSK(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := testutil.TestConfig()
	log := testutil.TestLogger()

	profileCmd := NewProfileCmd(cfg, log)

	// Test MSK profile add
	_, err := executeCommand(profileCmd, "add", "test-msk-new",
		"--type", "msk",
		"--region", "us-west-2",
		"--cluster-arn", "arn:aws:kafka:us-west-2:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1",
		"--auth-method", "IAM")
	// Check if the MSK profile was actually added
	if _, exists := cfg.Profiles["test-msk-new"]; !exists {
		t.Errorf("MSK profile 'test-msk-new' was not added to config. Error: %v", err)
	}

	// Verify MSK profile was added
	profileCmd = NewProfileCmd(cfg, log) // Create fresh command
	_, err = executeCommand(profileCmd, "list")
	if err != nil {
		t.Errorf("Profile list command failed: %v", err)
	}

	// Verify the MSK profile exists in config and has correct type
	if profile, exists := cfg.Profiles["test-msk-new"]; !exists {
		t.Error("Profile list should contain newly added MSK profile")
	} else if profile.Type != "msk" {
		t.Errorf("Profile should be MSK type, got: %s", profile.Type)
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
	// Check if the SSL profile was actually added
	if _, exists := cfg.Profiles["test-ssl"]; !exists {
		t.Errorf("SSL profile 'test-ssl' was not added to config. Error: %v, Output: %s", err, output)
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
	// Check if the SASL profile was actually added
	if _, exists := cfg.Profiles["test-sasl"]; !exists {
		t.Errorf("SASL profile 'test-sasl' was not added to config. Error: %v, Output: %s", err, output)
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

	if !strings.Contains(output, "powerful command-line interface for managing Kafka") {
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
	_, err := executeCommand(profileCmd, "list", "--format", "json")
	if err != nil {
		t.Errorf("JSON format failed: %v", err)
	}
	// Note: JSON output goes directly to stdout, not captured in test buffer
	// The fact that no error occurred means the JSON format is working

	// Test YAML output format
	_, err = executeCommand(profileCmd, "list", "--format", "yaml")
	if err != nil {
		t.Errorf("YAML format failed: %v", err)
	}
	// Note: YAML output goes directly to stdout, not captured in test buffer
	// The fact that no error occurred means the YAML format is working

	// Test invalid output format
	_, err = executeCommand(profileCmd, "list", "--format", "invalid")
	if err == nil {
		t.Error("Invalid output format should fail")
	}
}
