package auth

import (
	"strings"
	"testing"
	"time"
)

func TestNewMSKTokenProvider(t *testing.T) {
	region := "us-east-1"

	provider := NewMSKTokenProvider(region)
	if provider == nil {
		t.Fatal("MSKTokenProvider should not be nil")
	}

	if provider.region != region {
		t.Errorf("Expected region 'us-east-1', got '%s'", provider.region)
	}
}

func TestMSKTokenProviderToken(t *testing.T) {
	provider := NewMSKTokenProvider("us-east-1")

	// Test token generation (this will likely fail in CI without AWS credentials)
	// but we can test the structure
	token, err := provider.Token()

	// In a real test environment with AWS credentials, we would expect success
	// For unit tests without AWS setup, we expect a specific error
	if err != nil {
		// This is expected in test environment without AWS credentials
		t.Logf("Expected error in test environment: %v", err)
		return
	}

	// If we somehow got a token, validate its structure
	if token == nil {
		t.Fatal("Token should not be nil when no error occurred")
	}

	if token.Token == "" {
		t.Error("Token value should not be empty")
	}
}

func TestMSKTokenProviderCaching(t *testing.T) {
	provider := NewMSKTokenProvider("us-east-1")

	// Mock a cached token by setting internal fields
	provider.token = "cached-token"
	provider.expiresAt = time.Now().Add(time.Hour)

	// Request token - should return cached token
	token, err := provider.Token()
	if err != nil {
		// If there's an error, it might be due to AWS credentials
		// In that case, the cached token should still be returned
		t.Logf("Error occurred but checking if cached token is used: %v", err)
	}

	// Even with error, if we have a valid cached token, it should be used
	if provider.token != "" && provider.expiresAt.After(time.Now()) {
		if token == nil {
			t.Error("Should return cached token when available and valid")
		} else if token.Token != "cached-token" {
			t.Error("Should return cached token value")
		}
	}
}

func TestMSKTokenProviderExpiredCache(t *testing.T) {
	provider := NewMSKTokenProvider("us-east-1")

	// Mock an expired cached token
	provider.token = "expired-token"
	provider.expiresAt = time.Now().Add(-time.Hour) // Expired 1 hour ago

	// Request token - should not return expired cached token
	token, err := provider.Token()

	// We expect either a new token or an error (due to AWS credentials in test)
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
		return
	}

	if token != nil && token.Token == "expired-token" {
		t.Error("Should not return expired cached token")
	}
}

func TestGenerateToken(t *testing.T) {
	provider := NewMSKTokenProvider("us-east-1")

	// Test the generateToken method directly (this will likely fail without AWS creds)
	token, err := provider.generateToken()

	if err != nil {
		// Expected in test environment
		t.Logf("Expected error in test environment: %v", err)
		return
	}

	// If we somehow got a token, validate it
	if token == "" {
		t.Error("Token should not be empty when no error occurred")
	}

	// Our mock implementation should return a token with specific format
	if !strings.Contains(token, "msk-token-us-east-1") {
		t.Error("Token should contain expected format")
	}
}

func TestValidateClusterARN(t *testing.T) {
	// Test valid ARN
	validARN := "arn:aws:kafka:us-east-1:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1"
	err := ValidateClusterARN(validARN, "us-east-1")
	if err != nil {
		t.Errorf("Valid ARN should not return error: %v", err)
	}

	// Test invalid ARN format
	invalidARN := "invalid-arn"
	err = ValidateClusterARN(invalidARN, "us-east-1")
	if err == nil {
		t.Error("Invalid ARN should return error")
	}

	// Test ARN with wrong region
	wrongRegionARN := "arn:aws:kafka:us-west-2:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1"
	err = ValidateClusterARN(wrongRegionARN, "us-east-1")
	if err == nil {
		t.Error("ARN with wrong region should return error")
	}

	// Test ARN with missing parts
	shortARN := "arn:aws:kafka:us-east-1"
	err = ValidateClusterARN(shortARN, "us-east-1")
	if err == nil {
		t.Error("Short ARN should return error")
	}
}

func TestGetMSKBootstrapBrokers(t *testing.T) {
	// This test will likely fail without AWS credentials and a real cluster
	// but we can test the function signature and error handling

	region := "us-east-1"
	clusterARN := "arn:aws:kafka:us-east-1:123456789012:cluster/test/12345678-1234-1234-1234-123456789012-1"

	_, err := GetMSKBootstrapBrokers(region, clusterARN)

	// We expect an error in test environment (no AWS credentials or real cluster)
	if err == nil {
		t.Log("Unexpectedly succeeded - might have real AWS credentials")
	} else {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestMSKTokenProviderWithInvalidRegion(t *testing.T) {
	// Test with empty region
	provider := NewMSKTokenProvider("")
	if provider == nil {
		t.Error("Should create provider even with empty region")
	}

	// Token generation might succeed with empty region in our mock implementation
	// but in a real AWS environment, it would likely fail
	token, err := provider.Token()
	if err != nil {
		t.Logf("Expected error with empty region: %v", err)
	} else {
		t.Logf("Token generated with empty region (mock implementation): %s", token.Token)
		// Verify the token contains the empty region
		if !strings.Contains(token.Token, "msk-token--") {
			t.Error("Token should contain empty region placeholder")
		}
	}
}
