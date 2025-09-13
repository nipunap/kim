package cmd

import (
	"fmt"
	"strings"

	"github.com/nipunap/kim/internal/config"
	"github.com/nipunap/kim/internal/logger"
	"github.com/nipunap/kim/internal/ui"
	"github.com/nipunap/kim/pkg/types"

	"github.com/spf13/cobra"
)

// NewProfileCmd creates the profile command
func NewProfileCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage Kafka cluster profiles",
		Long:  "Commands for managing Kafka cluster profiles including adding, listing, and switching profiles.",
	}

	cmd.AddCommand(NewProfileListCmd(cfg, log))
	cmd.AddCommand(NewProfileAddCmd(cfg, log))
	cmd.AddCommand(NewProfileUseCmd(cfg, log))
	cmd.AddCommand(NewProfileDeleteCmd(cfg, log))

	return cmd
}

// NewProfileListCmd creates the profile list command
func NewProfileListCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		Long:  "List all configured Kafka cluster profiles.",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles := make([]*types.ProfileInfo, 0, len(cfg.Profiles))

			for name, profile := range cfg.Profiles {
				profileInfo := &types.ProfileInfo{
					Name:   name,
					Type:   profile.Type,
					Active: name == cfg.ActiveProfile,
				}

				// Add connection details based on type
				switch profile.Type {
				case "msk":
					profileInfo.Details = fmt.Sprintf("Region: %s", profile.Region)
				case "kafka":
					profileInfo.Details = fmt.Sprintf("Servers: %s", profile.BootstrapServers)
				}

				profiles = append(profiles, profileInfo)
			}

			displayOpts := &types.DisplayOptions{
				Format: format,
			}

			return ui.DisplayProfileList(profiles, displayOpts)
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format (table, json, yaml)")

	return cmd
}

// NewProfileAddCmd creates the profile add command
func NewProfileAddCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		profileType      string
		bootstrapServers string
		region           string
		clusterARN       string
		authMethod       string
		securityProtocol string
		saslMechanism    string
		saslUsername     string
		saslPassword     string
		sslCAFile        string
		sslCertFile      string
		sslKeyFile       string
		sslPassword      string
		sslCheckHostname bool
	)

	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a new profile",
		Long:  "Add a new Kafka cluster profile with the specified configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]


			// Check if profile already exists
			if _, exists := cfg.Profiles[name]; exists {
				return fmt.Errorf("profile '%s' already exists", name)
			}

			// Create profile based on type
			profile := &config.Profile{
				Name: name,
				Type: profileType,
			}

			switch profileType {
			case "msk":
				if region == "" {
					return fmt.Errorf("region is required for MSK profiles")
				}
				if clusterARN == "" {
					return fmt.Errorf("cluster-arn is required for MSK profiles")
				}

				profile.Region = region
				profile.ClusterARN = clusterARN
				profile.AuthMethod = authMethod
				if profile.AuthMethod == "" {
					profile.AuthMethod = "IAM" // Default to IAM
				}

			case "kafka":
				if bootstrapServers == "" {
					return fmt.Errorf("bootstrap-servers is required for Kafka profiles")
				}

				profile.BootstrapServers = bootstrapServers
				profile.SecurityProtocol = securityProtocol
				profile.SASLMechanism = saslMechanism
				profile.SASLUsername = saslUsername
				profile.SASLPassword = saslPassword
				profile.SSLCAFile = sslCAFile
				profile.SSLCertFile = sslCertFile
				profile.SSLKeyFile = sslKeyFile
				profile.SSLPassword = sslPassword
				profile.SSLCheckHostname = sslCheckHostname

			default:
				return fmt.Errorf("invalid profile type: %s (must be 'kafka' or 'msk')", profileType)
			}

			// Add profile
			if err := cfg.AddProfile(profile); err != nil {
				return fmt.Errorf("failed to add profile: %w", err)
			}

			fmt.Printf("Profile '%s' added successfully\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&profileType, "type", "", "profile type (kafka or msk)")
	cmd.Flags().StringVar(&bootstrapServers, "bootstrap-servers", "", "Kafka bootstrap servers (comma-separated)")
	cmd.Flags().StringVar(&region, "region", "", "AWS region for MSK")
	cmd.Flags().StringVar(&clusterARN, "cluster-arn", "", "MSK cluster ARN")
	cmd.Flags().StringVar(&authMethod, "auth-method", "IAM", "MSK authentication method (IAM or SASL_SCRAM)")
	cmd.Flags().StringVar(&securityProtocol, "security-protocol", "PLAINTEXT", "security protocol (PLAINTEXT, SSL, SASL_PLAINTEXT, SASL_SSL)")
	cmd.Flags().StringVar(&saslMechanism, "sasl-mechanism", "", "SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, GSSAPI)")
	cmd.Flags().StringVar(&saslUsername, "sasl-username", "", "SASL username")
	cmd.Flags().StringVar(&saslPassword, "sasl-password", "", "SASL password")
	cmd.Flags().StringVar(&sslCAFile, "ssl-ca-file", "", "SSL CA certificate file")
	cmd.Flags().StringVar(&sslCertFile, "ssl-cert-file", "", "SSL client certificate file")
	cmd.Flags().StringVar(&sslKeyFile, "ssl-key-file", "", "SSL client key file")
	cmd.Flags().StringVar(&sslPassword, "ssl-password", "", "SSL key password")
	cmd.Flags().BoolVar(&sslCheckHostname, "ssl-check-hostname", false, "enable SSL hostname verification")

	cmd.MarkFlagRequired("type")

	return cmd
}

// NewProfileUseCmd creates the profile use command
func NewProfileUseCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use NAME",
		Short: "Switch to a profile",
		Long:  "Switch to the specified profile as the active profile.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Check if profile exists
			if _, exists := cfg.Profiles[name]; !exists {
				return fmt.Errorf("profile '%s' does not exist", name)
			}

			// Set active profile
			if err := cfg.SetActiveProfile(name); err != nil {
				return fmt.Errorf("failed to set active profile: %w", err)
			}

			fmt.Printf("Switched to profile '%s'\n", name)
			return nil
		},
	}

	return cmd
}

// NewProfileDeleteCmd creates the profile delete command
func NewProfileDeleteCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a profile",
		Long:  "Delete the specified profile from the configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Check if profile exists
			if _, exists := cfg.Profiles[name]; !exists {
				return fmt.Errorf("profile '%s' does not exist", name)
			}

			// Prevent deletion of active profile without confirmation
			if name == cfg.ActiveProfile && !force {
				fmt.Printf("Profile '%s' is currently active. Are you sure you want to delete it? (y/N): ", name)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("Profile deletion cancelled")
					return nil
				}
			}

			// Delete profile
			delete(cfg.Profiles, name)

			// Clear active profile if it was the deleted one
			if name == cfg.ActiveProfile {
				cfg.ActiveProfile = ""
			}

			// Save configuration
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Printf("Profile '%s' deleted successfully\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}
