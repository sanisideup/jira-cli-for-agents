package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the Jira CLI configuration
type Config struct {
	Domain            string            `yaml:"domain"`                        // e.g., "yourcompany.atlassian.net"
	Email             string            `yaml:"email"`                         // User email for API token
	APIToken          string            `yaml:"api_token,omitempty"`           // Jira API token (deprecated: use keyring)
	DefaultProject    string            `yaml:"default_project,omitempty"`     // Optional default project key
	FieldMappings     map[string]string `yaml:"field_mappings,omitempty"`      // Custom field ID to name mappings
	MaxAttachmentSize int64             `yaml:"max_attachment_size,omitempty"` // Max attachment size in MB (default: 10)
	DownloadPath      string            `yaml:"download_path,omitempty"`       // Default download directory
	KeyringBackend    string            `yaml:"keyring_backend,omitempty"`     // Credential storage: auto, keychain, file
	UseKeyring        bool              `yaml:"use_keyring,omitempty"`         // Whether to use keyring for API token
	TemplatesDir      string            `yaml:"templates_dir,omitempty"`       // Custom templates directory path
}

const (
	// ConfigDirName is the name of the config directory
	ConfigDirName = ".jcfa"
	// ConfigFileName is the name of the config file
	ConfigFileName = "config.yaml"
	// ConfigFilePerms is the file permission for the config file (read/write for owner only)
	ConfigFilePerms = 0600
	// ConfigDirPerms is the directory permission for the config directory
	ConfigDirPerms = 0700
)

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDirName, ConfigFileName), nil
}

// GetConfigDir returns the full path to the config directory
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDirName), nil
}

// Load reads the config file from the default location and returns a Config struct
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFromPath(configPath)
}

// LoadFromPath reads the config file from a specific path and returns a Config struct
func LoadFromPath(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s. Run 'jcfa configure' to set up", configPath)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// LoadOrDefault loads the config file or returns an empty config if not found
func LoadOrDefault() *Config {
	config, err := Load()
	if err != nil {
		return &Config{
			FieldMappings: make(map[string]string),
		}
	}
	return config
}

// Save writes the config to the config file
func (c *Config) Save() error {
	// Validate before saving
	if err := c.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid config: %w", err)
	}

	// Ensure config directory exists
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, ConfigDirPerms); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Get config file path
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Write config file with restricted permissions
	if err := os.WriteFile(configPath, data, ConfigFilePerms); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the config has all required fields
func (c *Config) Validate() error {
	if c.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if c.Email == "" {
		return fmt.Errorf("email is required")
	}
	// API token can be empty if using keyring
	if c.APIToken == "" && !c.UseKeyring {
		return fmt.Errorf("api_token is required (or enable use_keyring)")
	}
	return nil
}

// GetAPIToken returns the API token, retrieving from keyring if configured
func (c *Config) GetAPIToken() string {
	// If UseKeyring is enabled and APIToken is empty, caller should use secrets package
	// This method returns what's in config for backward compatibility
	return c.APIToken
}

// GetBaseURL returns the full Jira API base URL
func (c *Config) GetBaseURL() string {
	return fmt.Sprintf("https://%s/rest/api/3", c.Domain)
}
