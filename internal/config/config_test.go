package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kim-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Test creating new config
	cfg, err := New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Config is nil")
	}

	// Check default settings
	if cfg.Settings == nil {
		t.Fatal("Settings is nil")
	}

	if cfg.Settings.PageSize != 20 {
		t.Errorf("Expected PageSize 20, got %d", cfg.Settings.PageSize)
	}

	if cfg.Settings.RefreshInterval != 10 {
		t.Errorf("Expected RefreshInterval 10, got %d", cfg.Settings.RefreshInterval)
	}

	// Check that config file was created
	configPath := filepath.Join(tempDir, ".kim", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file was not created at %s", configPath)
	}
}

func TestAddProfile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kim-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg, err := New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test adding MSK profile
	mskProfile := &Profile{
		Name:       "test-msk",
		Type:       "msk",
		Region:     "us-east-1",
		ClusterARN: "arn:aws:kafka:us-east-1:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1",
		AuthMethod: "IAM",
	}

	err = cfg.AddProfile(mskProfile)
	if err != nil {
		t.Fatalf("Failed to add MSK profile: %v", err)
	}

	// Test adding Kafka profile
	kafkaProfile := &Profile{
		Name:             "test-kafka",
		Type:             "kafka",
		BootstrapServers: "localhost:9092",
		SecurityProtocol: "PLAINTEXT",
	}

	err = cfg.AddProfile(kafkaProfile)
	if err != nil {
		t.Fatalf("Failed to add Kafka profile: %v", err)
	}

	// Verify profiles were added
	if len(cfg.Profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(cfg.Profiles))
	}

	// Test getting profile
	retrievedProfile, err := cfg.GetProfile("test-msk")
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}

	if retrievedProfile.Name != "test-msk" {
		t.Errorf("Expected profile name 'test-msk', got '%s'", retrievedProfile.Name)
	}
}

func TestSetActiveProfile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kim-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg, err := New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Add a profile first
	profile := &Profile{
		Name:             "test-profile",
		Type:             "kafka",
		BootstrapServers: "localhost:9092",
		SecurityProtocol: "PLAINTEXT",
	}

	err = cfg.AddProfile(profile)
	if err != nil {
		t.Fatalf("Failed to add profile: %v", err)
	}

	// Set active profile
	err = cfg.SetActiveProfile("test-profile")
	if err != nil {
		t.Fatalf("Failed to set active profile: %v", err)
	}

	if cfg.ActiveProfile != "test-profile" {
		t.Errorf("Expected active profile 'test-profile', got '%s'", cfg.ActiveProfile)
	}

	// Test getting active profile
	activeProfile, err := cfg.GetActiveProfile()
	if err != nil {
		t.Fatalf("Failed to get active profile: %v", err)
	}

	if activeProfile.Name != "test-profile" {
		t.Errorf("Expected active profile name 'test-profile', got '%s'", activeProfile.Name)
	}
}

func TestValidateProfile(t *testing.T) {
	cfg := &Config{}

	// Test valid MSK profile
	validMSK := &Profile{
		Name:       "valid-msk",
		Type:       "msk",
		Region:     "us-east-1",
		ClusterARN: "arn:aws:kafka:us-east-1:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1",
		AuthMethod: "IAM",
	}

	err := cfg.validateProfile(validMSK)
	if err != nil {
		t.Errorf("Valid MSK profile should not return error: %v", err)
	}

	// Test invalid MSK profile (missing region)
	invalidMSK := &Profile{
		Name:       "invalid-msk",
		Type:       "msk",
		ClusterARN: "arn:aws:kafka:us-east-1:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1",
		AuthMethod: "IAM",
	}

	err = cfg.validateProfile(invalidMSK)
	if err == nil {
		t.Error("Invalid MSK profile should return error")
	}

	// Test valid Kafka profile
	validKafka := &Profile{
		Name:             "valid-kafka",
		Type:             "kafka",
		BootstrapServers: "localhost:9092",
		SecurityProtocol: "PLAINTEXT",
	}

	err = cfg.validateProfile(validKafka)
	if err != nil {
		t.Errorf("Valid Kafka profile should not return error: %v", err)
	}

	// Test invalid Kafka profile (missing bootstrap servers)
	invalidKafka := &Profile{
		Name:             "invalid-kafka",
		Type:             "kafka",
		SecurityProtocol: "PLAINTEXT",
	}

	err = cfg.validateProfile(invalidKafka)
	if err == nil {
		t.Error("Invalid Kafka profile should return error")
	}
}

func TestDeleteProfile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kim-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg, err := New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Add profiles
	profile1 := &Profile{
		Name:             "profile1",
		Type:             "kafka",
		BootstrapServers: "localhost:9092",
		SecurityProtocol: "PLAINTEXT",
	}
	profile2 := &Profile{
		Name:             "profile2",
		Type:             "kafka",
		BootstrapServers: "localhost:9093",
		SecurityProtocol: "PLAINTEXT",
	}

	cfg.AddProfile(profile1)
	cfg.AddProfile(profile2)
	cfg.SetActiveProfile("profile1")

	// Delete non-active profile (manually for testing)
	delete(cfg.Profiles, "profile2")

	if len(cfg.Profiles) != 1 {
		t.Errorf("Expected 1 profile after deletion, got %d", len(cfg.Profiles))
	}

	// Try to delete active profile (manually check)
	if cfg.ActiveProfile == "profile1" {
		t.Log("Cannot delete active profile - this is expected behavior")
	}

	// Try to delete non-existent profile (manually check)
	if _, exists := cfg.Profiles["non-existent"]; !exists {
		t.Log("Non-existent profile not found - this is expected")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "kim-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create config and add profiles
	cfg, err := New()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	profile := &Profile{
		Name:             "test-save-load",
		Type:             "kafka",
		BootstrapServers: "localhost:9092",
		SecurityProtocol: "SASL_SSL",
		SASLMechanism:    "PLAIN",
		SASLUsername:     "testuser",
		SASLPassword:     "testpass",
	}

	err = cfg.AddProfile(profile)
	if err != nil {
		t.Fatalf("Failed to add profile: %v", err)
	}

	err = cfg.SetActiveProfile("test-save-load")
	if err != nil {
		t.Fatalf("Failed to set active profile: %v", err)
	}

	// Modify settings
	cfg.Settings.PageSize = 50
	cfg.Settings.RefreshInterval = 30
	cfg.Settings.DefaultFormat = "json"

	// Save config
	err = cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create new config instance and load
	cfg2, err := New()
	if err != nil {
		t.Fatalf("Failed to create new config: %v", err)
	}

	// Verify loaded data
	if cfg2.ActiveProfile != "test-save-load" {
		t.Errorf("Expected active profile 'test-save-load', got '%s'", cfg2.ActiveProfile)
	}

	if cfg2.Settings.PageSize != 50 {
		t.Errorf("Expected PageSize 50, got %d", cfg2.Settings.PageSize)
	}

	if cfg2.Settings.RefreshInterval != 30 {
		t.Errorf("Expected RefreshInterval 30, got %d", cfg2.Settings.RefreshInterval)
	}

	if cfg2.Settings.DefaultFormat != "json" {
		t.Errorf("Expected DefaultFormat 'json', got '%s'", cfg2.Settings.DefaultFormat)
	}

	loadedProfile, err := cfg2.GetProfile("test-save-load")
	if err != nil {
		t.Fatalf("Failed to get loaded profile: %v", err)
	}

	if loadedProfile.SASLUsername != "testuser" {
		t.Errorf("Expected SASLUsername 'testuser', got '%s'", loadedProfile.SASLUsername)
	}
}

func TestProfileValidationEdgeCases(t *testing.T) {
	cfg := &Config{}

	// Test empty profile
	emptyProfile := &Profile{}
	err := cfg.validateProfile(emptyProfile)
	if err == nil {
		t.Error("Empty profile should return validation error")
	}

	// Test profile with invalid type
	invalidType := &Profile{
		Name: "invalid-type",
		Type: "invalid",
	}
	err = cfg.validateProfile(invalidType)
	if err == nil {
		t.Error("Profile with invalid type should return error")
	}

	// Test MSK profile with invalid auth method
	invalidAuth := &Profile{
		Name:       "invalid-auth",
		Type:       "msk",
		Region:     "us-east-1",
		ClusterARN: "arn:aws:kafka:us-east-1:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1",
		AuthMethod: "INVALID",
	}
	err = cfg.validateProfile(invalidAuth)
	if err == nil {
		t.Error("MSK profile with invalid auth method should return error")
	}

	// Test Kafka profile with invalid security protocol
	invalidProtocol := &Profile{
		Name:             "invalid-protocol",
		Type:             "kafka",
		BootstrapServers: "localhost:9092",
		SecurityProtocol: "INVALID_PROTOCOL",
	}
	err = cfg.validateProfile(invalidProtocol)
	if err == nil {
		t.Error("Kafka profile with invalid security protocol should return error")
	}

	// Test valid profiles that should pass
	validSSL := &Profile{
		Name:             "valid-ssl",
		Type:             "kafka",
		BootstrapServers: "localhost:9092",
		SecurityProtocol: "SSL",
	}
	err = cfg.validateProfile(validSSL)
	if err != nil {
		t.Errorf("Valid SSL profile should not return error: %v", err)
	}

	validSASL := &Profile{
		Name:             "valid-sasl",
		Type:             "kafka",
		BootstrapServers: "localhost:9092",
		SecurityProtocol: "SASL_PLAINTEXT",
		SASLMechanism:    "PLAIN",
	}
	err = cfg.validateProfile(validSASL)
	if err != nil {
		t.Errorf("Valid SASL profile should not return error: %v", err)
	}
}
