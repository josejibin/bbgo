package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Workspace   string `yaml:"workspace"`
	DefaultRepo string `yaml:"default_repo,omitempty"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".bbgo", "config.yaml"), nil
}

func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}
