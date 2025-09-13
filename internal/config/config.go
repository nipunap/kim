package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Profiles      map[string]*Profile `mapstructure:"profiles" yaml:"profiles"`
	ActiveProfile string              `mapstructure:"active_profile" yaml:"active_profile"`
	Settings      *Settings           `mapstructure:"settings" yaml:"settings"`
	configPath    string
}

// Profile represents a Kafka cluster configuration
type Profile struct {
	Name             string            `mapstructure:"name" yaml:"name"`
	Type             string            `mapstructure:"type" yaml:"type"` // "kafka" or "msk"
	BootstrapServers string            `mapstructure:"bootstrap_servers,omitempty" yaml:"bootstrap_servers,omitempty"`
	Region           string            `mapstructure:"region,omitempty" yaml:"region,omitempty"`
	ClusterARN       string            `mapstructure:"cluster_arn,omitempty" yaml:"cluster_arn,omitempty"`
	AuthMethod       string            `mapstructure:"auth_method,omitempty" yaml:"auth_method,omitempty"`
	SecurityProtocol string            `mapstructure:"security_protocol,omitempty" yaml:"security_protocol,omitempty"`
	SASLMechanism    string            `mapstructure:"sasl_mechanism,omitempty" yaml:"sasl_mechanism,omitempty"`
	SASLUsername     string            `mapstructure:"sasl_username,omitempty" yaml:"sasl_username,omitempty"`
	SASLPassword     string            `mapstructure:"sasl_password,omitempty" yaml:"sasl_password,omitempty"`
	SSLCAFile        string            `mapstructure:"ssl_ca_file,omitempty" yaml:"ssl_ca_file,omitempty"`
	SSLCertFile      string            `mapstructure:"ssl_cert_file,omitempty" yaml:"ssl_cert_file,omitempty"`
	SSLKeyFile       string            `mapstructure:"ssl_key_file,omitempty" yaml:"ssl_key_file,omitempty"`
	SSLPassword      string            `mapstructure:"ssl_password,omitempty" yaml:"ssl_password,omitempty"`
	SSLCheckHostname bool              `mapstructure:"ssl_check_hostname,omitempty" yaml:"ssl_check_hostname,omitempty"`
	Extra            map[string]string `mapstructure:"extra,omitempty" yaml:"extra,omitempty"`
}

// Settings represents application settings
type Settings struct {
	PageSize        int    `mapstructure:"page_size" yaml:"page_size"`
	RefreshInterval int    `mapstructure:"refresh_interval" yaml:"refresh_interval"`
	DefaultFormat   string `mapstructure:"default_format" yaml:"default_format"`
	ColorScheme     string `mapstructure:"color_scheme" yaml:"color_scheme"`
	VimMode         bool   `mapstructure:"vim_mode" yaml:"vim_mode"`
}

// New creates a new configuration instance
func New() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".kim")
	configPath := filepath.Join(configDir, "config.yaml")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Initialize viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	// Set defaults
	viper.SetDefault("profiles", map[string]*Profile{})
	viper.SetDefault("active_profile", "")
	viper.SetDefault("settings", &Settings{
		PageSize:        20,
		RefreshInterval: 10,
		DefaultFormat:   "table",
		ColorScheme:     "default",
		VimMode:         true,
	})

	config := &Config{
		configPath: configPath,
	}

	// Try to read existing config
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create default
			if err := config.createDefaultConfig(); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			// Try to read the newly created config
			if err := viper.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read newly created config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal config
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

// createDefaultConfig creates a default configuration file
func (c *Config) createDefaultConfig() error {
	c.Profiles = make(map[string]*Profile)
	c.ActiveProfile = ""
	c.Settings = &Settings{
		PageSize:        20,
		RefreshInterval: 10,
		DefaultFormat:   "table",
		ColorScheme:     "default",
		VimMode:         true,
	}

	return c.Save()
}

// Save saves the configuration to file
func (c *Config) Save() error {
	viper.Set("profiles", c.Profiles)
	viper.Set("active_profile", c.ActiveProfile)
	viper.Set("settings", c.Settings)

	// Try WriteConfig first, if it fails (file doesn't exist), use WriteConfigAs
	if err := viper.WriteConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return viper.WriteConfigAs(c.configPath)
		}
		return err
	}
	return nil
}

// AddProfile adds a new profile to the configuration
func (c *Config) AddProfile(profile *Profile) error {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}

	// Validate profile
	if err := c.validateProfile(profile); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	c.Profiles[profile.Name] = profile
	return c.Save()
}

// GetProfile returns a profile by name
func (c *Config) GetProfile(name string) (*Profile, error) {
	profile, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}
	return profile, nil
}

// GetActiveProfile returns the currently active profile
func (c *Config) GetActiveProfile() (*Profile, error) {
	if c.ActiveProfile == "" {
		return nil, fmt.Errorf("no active profile set")
	}
	return c.GetProfile(c.ActiveProfile)
}

// SetActiveProfile sets the active profile
func (c *Config) SetActiveProfile(name string) error {
	if _, exists := c.Profiles[name]; !exists {
		return fmt.Errorf("profile '%s' not found", name)
	}
	c.ActiveProfile = name
	return c.Save()
}

// ListProfiles returns all profile names
func (c *Config) ListProfiles() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

// validateProfile validates a profile configuration
func (c *Config) validateProfile(profile *Profile) error {
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}

	switch profile.Type {
	case "msk":
		if profile.Region == "" {
			return fmt.Errorf("region is required for MSK profiles")
		}
		if profile.ClusterARN == "" {
			return fmt.Errorf("cluster_arn is required for MSK profiles")
		}
		if profile.AuthMethod != "" && profile.AuthMethod != "IAM" && profile.AuthMethod != "SASL_SCRAM" {
			return fmt.Errorf("auth_method must be either 'IAM' or 'SASL_SCRAM' for MSK profiles")
		}
	case "kafka":
		if profile.BootstrapServers == "" {
			return fmt.Errorf("bootstrap_servers is required for Kafka profiles")
		}
		if profile.SecurityProtocol != "" {
			validProtocols := []string{"PLAINTEXT", "SSL", "SASL_PLAINTEXT", "SASL_SSL"}
			valid := false
			for _, p := range validProtocols {
				if profile.SecurityProtocol == p {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid security_protocol: %s", profile.SecurityProtocol)
			}
		}
	case "":
		return fmt.Errorf("profile type is required (must be 'kafka' or 'msk')")
	default:
		return fmt.Errorf("invalid profile type: %s (must be 'kafka' or 'msk')", profile.Type)
	}

	return nil
}
