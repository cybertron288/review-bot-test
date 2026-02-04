package config

import "time"

// Default configuration values
const (
	DefaultClaudeCodeCmd         = "claude"
	DefaultOpenCodeCmd           = "opencode"
	DefaultProcessTimeout        = 15 * time.Minute  // LLM calls can take a while
	DefaultHealthCheckInterval   = 5 * time.Minute   // Don't check too frequently
	DefaultPlansDir              = ".agentflow/plans"
	DefaultTheme                 = "default"
	DefaultAutoApprovePermissions = true
)

// Defaults returns a Config with all default values
func Defaults() *Config {
	return &Config{
		CLI: CLIConfig{
			ClaudeCode: DefaultClaudeCodeCmd,
			OpenCode:   DefaultOpenCodeCmd,
		},
		Process: ProcessConfig{
			Timeout:             DefaultProcessTimeout,
			HealthCheckInterval: DefaultHealthCheckInterval,
		},
		Permissions: PermissionsConfig{
			AutoApprove: DefaultAutoApprovePermissions,
		},
		Paths: PathsConfig{
			PlansDir: DefaultPlansDir,
		},
		TUI: TUIConfig{
			Theme:  DefaultTheme,
			Editor: "", // Will use $EDITOR
		},
	}
}

