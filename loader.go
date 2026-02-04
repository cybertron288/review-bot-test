package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the agentflow configuration
type Config struct {
	CLI         CLIConfig         `yaml:"cli"`
	Process     ProcessConfig     `yaml:"process"`
	Permissions PermissionsConfig `yaml:"permissions"`
	Paths       PathsConfig       `yaml:"paths"`
	TUI         TUIConfig         `yaml:"tui"`
}

// CLIConfig holds CLI tool paths
type CLIConfig struct {
	ClaudeCode  string `yaml:"claude_code"`
	OpenCode    string `yaml:"opencode"`
	ClaudeModel string `yaml:"claude_model"` // Model to use (e.g., "sonnet", "opus")
}

// ProcessConfig holds process management settings
type ProcessConfig struct {
	Timeout             time.Duration `yaml:"timeout"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
}

// PermissionsConfig holds permission settings
type PermissionsConfig struct {
	AutoApprove bool `yaml:"auto_approve"`
}

// PathsConfig holds path settings
type PathsConfig struct {
	PlansDir string `yaml:"plans_dir"`
}

// TUIConfig holds TUI settings
type TUIConfig struct {
	Theme  string `yaml:"theme"`
	Editor string `yaml:"editor"`
}

// Load loads the configuration from files with priority:
// 1. .agentflow/config.yaml (project-specific)
// 2. ~/.config/agentflow/config.yaml (user global)
// 3. Built-in defaults
func Load() (*Config, error) {
	cfg := Defaults()

	// Try user global config first
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(homeDir, ".config", "agentflow", "config.yaml")
		if err := loadFromFile(globalPath, cfg); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Try project-specific config (overrides global)
	projectPath := filepath.Join(".agentflow", "config.yaml")
	if err := loadFromFile(projectPath, cfg); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Apply environment variable for editor if not set
	if cfg.TUI.Editor == "" {
		cfg.TUI.Editor = os.Getenv("EDITOR")
		if cfg.TUI.Editor == "" {
			cfg.TUI.Editor = "vim" // fallback
		}
	}

	return cfg, nil
}

func loadFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, cfg)
}

// Save saves the configuration to the project-specific config file
func Save(cfg *Config) error {
	dir := filepath.Dir(filepath.Join(".agentflow", "config.yaml"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(".agentflow", "config.yaml"), data, 0644)
}

// GetPlansDir returns the absolute path to the plans directory
func (c *Config) GetPlansDir() (string, error) {
	if filepath.IsAbs(c.Paths.PlansDir) {
		return c.Paths.PlansDir, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(cwd, c.Paths.PlansDir), nil
}

