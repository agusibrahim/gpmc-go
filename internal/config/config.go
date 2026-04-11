package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	AuthData string `yaml:"auth_data"`
	Proxy    string `yaml:"proxy"`
	Language string `yaml:"language"`
	Timeout  int    `yaml:"timeout"`
	LogLevel string `yaml:"log_level"`
	Threads  int    `yaml:"threads"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		Timeout:  60,
		LogLevel: "INFO",
		Threads:  1,
	}
}

// Load loads configuration from a YAML file
func Load(configPath string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// Save saves configuration to a YAML file
func (c *Config) Save(configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultConfigPath returns the default config file path
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".gpmc", "config.yaml")
}

// LoadOrDefault loads config from file or returns default config
func LoadOrDefault(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = GetDefaultConfigPath()
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config
		return DefaultConfig(), nil
	}

	return Load(configPath)
}

// MergeWithDefaults merges config with environment variables and defaults
// Priority: CLI flags > env vars > config file > defaults
func (c *Config) MergeWithDefaults() *Config {
	config := DefaultConfig()

	// Apply config file values
	if c.AuthData != "" {
		config.AuthData = c.AuthData
	}
	if c.Proxy != "" {
		config.Proxy = c.Proxy
	}
	if c.Language != "" {
		config.Language = c.Language
	}
	if c.Timeout > 0 {
		config.Timeout = c.Timeout
	}
	if c.LogLevel != "" {
		config.LogLevel = c.LogLevel
	}
	if c.Threads > 0 {
		config.Threads = c.Threads
	}

	// Apply environment variables
	if envAuthData := os.Getenv("GP_AUTH_DATA"); envAuthData != "" {
		config.AuthData = envAuthData
	}

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Check if auth_data is set (either from config or env)
	if c.AuthData == "" && os.Getenv("GP_AUTH_DATA") == "" {
		return fmt.Errorf("auth_data is required (set in config file or GP_AUTH_DATA env var)")
	}

	// Validate other fields
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if c.Threads < 0 {
		return fmt.Errorf("threads must be positive")
	}

	return nil
}

// CreateTemplateConfig creates a template config file
func CreateTemplateConfig(configPath string) error {
	config := DefaultConfig()
	config.AuthData = "YOUR_AUTH_DATA_HERE"

	return config.Save(configPath)
}
