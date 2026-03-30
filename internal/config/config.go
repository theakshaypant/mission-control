package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = ".config/mission-control/config.yaml"

// AppConfig is the top-level configuration for mission-control.
type AppConfig struct {
	Sources []RawSourceConfig `yaml:"sources"`
}

// RawSourceConfig holds the common fields for any source plus a catch-all
// map for source-specific fields. The Extra map is used by each source's
// factory to unmarshal its own typed config.
type RawSourceConfig struct {
	Type string         `yaml:"type"`
	Name string         `yaml:"name"`
	Extra map[string]any `yaml:",inline"`
}

// DefaultConfigPath returns the default path to the config file.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	return filepath.Join(home, defaultConfigPath), nil
}

// Load reads and parses the config file at path.
func Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to path, creating parent directories as needed.
// The file is written with 0600 permissions (owner read/write only).
func Save(cfg *AppConfig, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}
