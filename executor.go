package claude

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ravi/agentflow/internal/config"
	"github.com/ravi/agentflow/internal/logger"
	"github.com/ravi/agentflow/internal/process"
)

// Executor handles Claude Code CLI execution
type Executor struct {
	config  *config.Config
	manager *process.Manager
}

// ExecutionResult holds the result of a Claude Code execution
type ExecutionResult struct {
	Output     string
	Tokens     int
	Duration   time.Duration
	Success    bool
	Error      error
}

// NewExecutor creates a new Claude Code executor
func NewExecutor(cfg *config.Config, pm *process.Manager) *Executor {
	return &Executor{
		config:  cfg,
		manager: pm,
	}
}

// Execute runs Claude Code with the given prompt
func (e *Executor) Execute(ctx context.Context, prompt string, systemPrompt string) (*ExecutionResult, error) {
	startTime := time.Now()
	result := &ExecutionResult{}

	// Build command arguments
	args := e.buildArgs(prompt, systemPrompt)
	logger.Info("Starting Claude Code with args: %v", args)

	// Start the process
	mp, err := e.manager.Start(ctx, e.config.CLI.ClaudeCode, args...)
	if err != nil {
		logger.Error("Failed to start Claude Code: %v", err)
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}
	logger.Info("Claude Code started with PID: %d", mp.PID)

	// Collect output
	var output strings.Builder
	err = process.StreamWithCallback(ctx, mp, func(line string) {
		output.WriteString(line)
		output.WriteString("\n")
		// Update last output time for health monitoring
		e.manager.UpdateLastOutput(mp.PID)
	})

	result.Duration = time.Since(startTime)
	result.Output = output.String()

	if err != nil {
		// Include output in error for better debugging
		if result.Output != "" {
			result.Error = fmt.Errorf("%w: %s", err, strings.TrimSpace(result.Output))
		} else {
			result.Error = err
		}
		result.Success = false
		return result, nil
	}

	// Wait for process to complete
	if waitErr := mp.Wait(); waitErr != nil {
		// Include output in error message for better debugging
		errMsg := waitErr.Error()
		if result.Output != "" {
			errMsg = strings.TrimSpace(result.Output)
		}
		logger.Error("Claude Code process failed: %s", errMsg)
		result.Error = fmt.Errorf("claude error: %s", errMsg)
		result.Success = false
		return result, nil
	}

	// Check for error patterns in output (e.g., rate limit)
	if hasError, errMsg := detectErrorInOutput(result.Output); hasError {
		logger.Error("Claude Code error detected in output: %s", errMsg)
		result.Error = fmt.Errorf("claude error: %s", errMsg)
		result.Success = false
		return result, nil
	}

	logger.Info("Claude Code completed successfully in %v", result.Duration)
	result.Success = true
	// TODO: Parse token count from output
	result.Tokens = estimateTokens(result.Output)

	return result, nil
}

// ExecuteWithStreaming runs Claude Code and streams output via callback
func (e *Executor) ExecuteWithStreaming(ctx context.Context, prompt string, systemPrompt string, callback func(line string)) (*ExecutionResult, error) {
	startTime := time.Now()
	result := &ExecutionResult{}

	// Build command arguments
	args := e.buildArgs(prompt, systemPrompt)

	// Start the process
	mp, err := e.manager.Start(ctx, e.config.CLI.ClaudeCode, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}
	logger.Info("Claude Code started with PID: %d", mp.PID)

	// Collect output while streaming
	var output strings.Builder
	lineCount := 0
	err = process.StreamWithCallback(ctx, mp, func(line string) {
		lineCount++
		logger.Debug("Executor received line %d: %s", lineCount, line)
		output.WriteString(line)
		output.WriteString("\n")
		callback(line)
		// Update last output time for health monitoring
		e.manager.UpdateLastOutput(mp.PID)
	})

	logger.Info("Streaming complete, received %d lines, stream error: %v", lineCount, err)
	result.Duration = time.Since(startTime)
	result.Output = output.String()
	logger.Debug("Total output length: %d bytes", len(result.Output))

	if err != nil {
		// Include output in error for better debugging
		if result.Output != "" {
			result.Error = fmt.Errorf("%w: %s", err, strings.TrimSpace(result.Output))
		} else {
			result.Error = err
		}
		result.Success = false
		return result, nil
	}

	// Wait for process to complete
	if waitErr := mp.Wait(); waitErr != nil {
		// Include output in error message for better debugging
		errMsg := waitErr.Error()
		if result.Output != "" {
			errMsg = strings.TrimSpace(result.Output)
		}
		result.Error = fmt.Errorf("claude error: %s", errMsg)
		result.Success = false
		return result, nil
	}

	// Check for error patterns in output (e.g., rate limit)
	if hasError, errMsg := detectErrorInOutput(result.Output); hasError {
		logger.Error("Claude Code error detected in output: %s", errMsg)
		result.Error = fmt.Errorf("claude error: %s", errMsg)
		result.Success = false
		return result, nil
	}

	result.Success = true
	result.Tokens = estimateTokens(result.Output)

	return result, nil
}

// buildArgs builds the command line arguments for Claude Code
func (e *Executor) buildArgs(prompt string, systemPrompt string) []string {
	args := []string{}

	// Skip permission prompts if configured
	if e.config.Permissions.AutoApprove {
		args = append(args, "--dangerously-skip-permissions")
	}

	// Print mode for non-interactive output
	args = append(args, "--print")

	// Use specific model if configured (models like "opus" may have higher output limits)
	if e.config.CLI.ClaudeModel != "" {
		args = append(args, "--model", e.config.CLI.ClaudeModel)
	}

	// Combine system prompt and user prompt into a single argument
	// Claude CLI format: claude --print "<combined_prompt>"
	var fullPrompt string
	if systemPrompt != "" {
		fullPrompt = systemPrompt + " " + prompt
	} else {
		fullPrompt = prompt
	}
	args = append(args, fullPrompt)

	return args
}

// estimateTokens provides a rough estimate of tokens used
// This is a simplistic estimation; actual token counts would come from the API
func estimateTokens(text string) int {
	// Rough estimation: ~4 characters per token for English text
	return len(text) / 4
}

// Known error patterns from Claude CLI
var errorPatterns = []string{
	"you've hit your limit",
	"rate limit",
	"api key",
	"authentication failed",
	"unauthorized",
	"quota exceeded",
	"invalid api key",
	"access denied",
	"permission denied",
	"not authenticated",
}

// detectErrorInOutput checks if the output contains known error patterns
func detectErrorInOutput(output string) (bool, string) {
	lowerOutput := strings.ToLower(output)
	for _, pattern := range errorPatterns {
		if strings.Contains(lowerOutput, pattern) {
			// Extract the relevant error line from original output
			for _, line := range strings.Split(output, "\n") {
				if strings.Contains(strings.ToLower(line), pattern) {
					return true, strings.TrimSpace(line)
				}
			}
			return true, pattern
		}
	}
	return false, ""
}

// Cancel cancels any running Claude Code process
func (e *Executor) Cancel() {
	e.manager.Cleanup()
}

