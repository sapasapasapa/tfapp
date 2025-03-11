// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	Colors ColorConfig `yaml:"colors"`
	UI     UIConfig    `yaml:"ui"`
}

// UIConfig holds the UI configuration values.
type UIConfig struct {
	// Type of spinner to use for loading animations
	// Available options: MiniDot, Dot, Line, Jump, Pulse, Points, Globe, Moon, Monkey, Meter
	// See full reference: https://pkg.go.dev/github.com/charmbracelet/bubbles@v0.20.0/spinner
	SpinnerType string `yaml:"spinner_type"`

	// Character to use for the cursor in menus (default: ">")
	CursorChar string `yaml:"cursor_char"`
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
			Info:      "#3366cc", // Cyan/Blue (was #36c)
			Success:   "#22aa22", // Green (was #2a2)
			Warning:   "#ffaa00", // Yellow/Orange (was #fa0)
			Error:     "#ff3333", // Red (was #f33)
			Highlight: "#8833ff", // Purple (was #83f)
			Faint:     "#777777", // Gray (was #777)
		},
		UI: UIConfig{
			SpinnerType: "MiniDot", // Default spinner type
			CursorChar:  ">",       // Default cursor character
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

	// Add comments for documentation
	yamlString := string(data)

	// Add spinner documentation
	yamlString = strings.Replace(yamlString,
		"ui:",
		`ui:
  # For spinner_type, available options are:
  # MiniDot, Dot, Line, Jump, Pulse, Points, Globe, Moon, Monkey, Meter
  # See: https://pkg.go.dev/github.com/charmbracelet/bubbles@v0.20.0/spinner`,
		1)

	// Write to file
	if err := os.WriteFile(filename, []byte(yamlString), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
