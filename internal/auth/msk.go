package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
)

// MSKTokenProvider implements sarama.TokenProvider for MSK IAM authentication
type MSKTokenProvider struct {
	region    string
	token     string
	expiresAt time.Time
	mutex     sync.RWMutex
}

// NewMSKTokenProvider creates a new MSK token provider
func NewMSKTokenProvider(region string) *MSKTokenProvider {
	return &MSKTokenProvider{
		region: region,
	}
}

// Token returns a valid MSK authentication token
func (p *MSKTokenProvider) Token() (*sarama.AccessToken, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Return cached token if still valid (with 1-minute buffer)
	if p.token != "" && time.Now().Before(p.expiresAt.Add(-time.Minute)) {
		return &sarama.AccessToken{
			Token: p.token,
		}, nil
	}

	// Generate new token
	token, err := p.generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate MSK token: %w", err)
	}

	p.token = token
	p.expiresAt = time.Now().Add(15 * time.Minute) // MSK tokens are valid for 15 minutes

	return &sarama.AccessToken{
		Token: token,
	}, nil
}

// generateToken generates a new MSK authentication token
func (p *MSKTokenProvider) generateToken() (string, error) {
	// Load AWS configuration
	_, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(p.region),
	)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// This is a simplified implementation. In a real implementation,
	// you would use the AWS MSK IAM SASL Signer library or implement
	// the AWS SigV4 signing process for MSK

	// For now, return a placeholder that indicates the token generation logic
	// In production, you would use:
	// - github.com/aws/aws-msk-iam-sasl-signer-go
	// - Or implement AWS SigV4 signing manually

	return fmt.Sprintf("msk-token-%s-%d", p.region, time.Now().Unix()), nil
}

// GetMSKBootstrapBrokers retrieves bootstrap brokers for an MSK cluster
func GetMSKBootstrapBrokers(region, clusterARN string) (string, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create MSK client
	client := kafka.NewFromConfig(cfg)

	// Get bootstrap brokers
	input := &kafka.GetBootstrapBrokersInput{
		ClusterArn: aws.String(clusterARN),
	}

	result, err := client.GetBootstrapBrokers(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("failed to get bootstrap brokers: %w", err)
	}

	// Prefer SASL_IAM brokers for IAM authentication
	if result.BootstrapBrokerStringSaslIam != nil {
		return *result.BootstrapBrokerStringSaslIam, nil
	}

	// Fall back to SASL_SCRAM if available
	if result.BootstrapBrokerStringSaslScram != nil {
		return *result.BootstrapBrokerStringSaslScram, nil
	}

	// Fall back to TLS
	if result.BootstrapBrokerStringTls != nil {
		return *result.BootstrapBrokerStringTls, nil
	}

	// Fall back to plaintext (not recommended for production)
	if result.BootstrapBrokerString != nil {
		return *result.BootstrapBrokerString, nil
	}

	return "", fmt.Errorf("no bootstrap brokers available for cluster %s", clusterARN)
}

// ValidateClusterARN validates the format of an MSK cluster ARN
func ValidateClusterARN(arn, region string) error {
	if !strings.HasPrefix(arn, "arn:aws:kafka:") {
		return fmt.Errorf("invalid MSK cluster ARN format")
	}

	// Extract region from ARN
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return fmt.Errorf("invalid MSK cluster ARN format")
	}

	arnRegion := parts[3]
	if arnRegion != region {
		return fmt.Errorf("ARN region (%s) does not match provided region (%s)", arnRegion, region)
	}

	return nil
}
