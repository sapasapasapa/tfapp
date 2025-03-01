// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	Colors ColorConfig `yaml:"colors"`
}

// ColorConfig holds the color configuration values.
type ColorConfig struct {
	Info      string `yaml:"info"`      // Informational messages (cyan/blue)
	Success   string `yaml:"success"`   // Success messages (green)
	Warning   string `yaml:"warning"`   // Warning messages (yellow/orange)
	Error     string `yaml:"error"`     // Error messages (red)
	Highlight string `yaml:"highlight"` // Highlighted elements (purple)
	Faint     string `yaml:"faint"`     // Less important text (gray)
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Colors: ColorConfig{
			Info:      "#36c", // Cyan/Blue
			Success:   "#2a2", // Green
			Warning:   "#fa0", // Yellow/Orange
			Error:     "#f33", // Red
			Highlight: "#83f", // Purple
			Faint:     "#777", // Gray
		},
	}
}

// ConfigFilePath returns the path to the configuration file.
func ConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "tfapp")
	return filepath.Join(configDir, "config.yaml"), nil
}

// LoadConfig loads the configuration from the config file.
// If the file doesn't exist, it creates a default configuration.
// Returns the config, a flag indicating if the config was created, and any error.
func LoadConfig() (*Config, bool, error) {
	filename, err := ConfigFilePath()
	if err != nil {
		return nil, false, err
	}

	// Check if the file exists
	configCreated := false
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// File doesn't exist, create the default one
		if err := createDefaultConfig(filename); err != nil {
			return nil, false, fmt.Errorf("failed to create default config: %w", err)
		}
		configCreated = true
	}

	// Read the config file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, configCreated, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, configCreated, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, configCreated, nil
}

// createDefaultConfig creates a default configuration file.
func createDefaultConfig(filename string) error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create the default config
	config := DefaultConfig()

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
