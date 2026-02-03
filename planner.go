package claude

import (
	"context"
	"fmt"
	"strings"

	"github.com/ravi/agentflow/internal/config"
	"github.com/ravi/agentflow/internal/plan"
	"github.com/ravi/agentflow/internal/process"
)

// Planner handles plan generation with Claude Code
type Planner struct {
	executor    *Executor
	planManager *plan.Manager
	config      *config.Config
}

// PlanResult holds the result of plan generation
type PlanResult struct {
	PlanInfo    *plan.PlanInfo
	ParsedPlan  *ParsedPlan
	RawOutput   string
	Tokens      int
	Success     bool
	Error       error
}

// NewPlanner creates a new planner
func NewPlanner(cfg *config.Config, pm *process.Manager, planMgr *plan.Manager) *Planner {
	return &Planner{
		executor:    NewExecutor(cfg, pm),
		planManager: planMgr,
		config:      cfg,
	}
}

// GeneratePlan generates a new plan for the given task
func (p *Planner) GeneratePlan(ctx context.Context, task string) (*PlanResult, error) {
	result := &PlanResult{}

	// Create a new plan
	planInfo, err := p.planManager.Create(task)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	result.PlanInfo = planInfo

	// Build the prompt
	prompt := p.buildPlanPrompt(task)

	// Execute Claude Code
	execResult, err := p.executor.Execute(ctx, prompt, plannerSystemPrompt)
	if err != nil {
		result.Error = err
		return result, nil
	}

	result.RawOutput = execResult.Output
	result.Tokens = execResult.Tokens

	if !execResult.Success {
		result.Error = execResult.Error
		return result, nil
	}

	// Check if output was truncated
	if IsPlanTruncated(execResult.Output) {
		result.Error = fmt.Errorf("plan output was truncated (Claude may have hit output token limit). Try breaking down your task into smaller parts")
		// Still try to parse what we got
		parsedPlan, _ := ParsePlanFromOutput(execResult.Output)
		if parsedPlan != nil {
			result.ParsedPlan = parsedPlan
			// Save partial plan with truncation warning
			p.planManager.SavePlan(planInfo, parsedPlan.RawContent+"\n\n⚠️ WARNING: This plan was truncated. Request a simpler task or iterate with feedback.")
		}
		return result, nil
	}

	// Parse the plan
	parsedPlan, err := ParsePlanFromOutput(execResult.Output)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse plan: %w", err)
		return result, nil
	}
	result.ParsedPlan = parsedPlan

	// Save the plan content
	if err := p.planManager.SavePlan(planInfo, parsedPlan.RawContent); err != nil {
		result.Error = fmt.Errorf("failed to save plan: %w", err)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// GeneratePlanWithStreaming generates a plan with streaming output
func (p *Planner) GeneratePlanWithStreaming(ctx context.Context, task string, callback func(line string)) (*PlanResult, error) {
	result := &PlanResult{}

	// Create a new plan
	planInfo, err := p.planManager.Create(task)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	result.PlanInfo = planInfo

	// Build the prompt
	prompt := p.buildPlanPrompt(task)

	// Execute Claude Code with streaming
	execResult, err := p.executor.ExecuteWithStreaming(ctx, prompt, plannerSystemPrompt, callback)
	if err != nil {
		result.Error = err
		return result, nil
	}

	result.RawOutput = execResult.Output
	result.Tokens = execResult.Tokens

	if !execResult.Success {
		result.Error = execResult.Error
		return result, nil
	}

	// Check if output was truncated
	if IsPlanTruncated(execResult.Output) {
		result.Error = fmt.Errorf("plan output was truncated (Claude may have hit output token limit). Try breaking down your task into smaller parts")
		// Still try to parse what we got
		parsedPlan, _ := ParsePlanFromOutput(execResult.Output)
		if parsedPlan != nil {
			result.ParsedPlan = parsedPlan
			// Save partial plan with truncation warning
			p.planManager.SavePlan(planInfo, parsedPlan.RawContent+"\n\n⚠️ WARNING: This plan was truncated. Request a simpler task or iterate with feedback.")
		}
		return result, nil
	}

	// Parse the plan
	parsedPlan, err := ParsePlanFromOutput(execResult.Output)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse plan: %w", err)
		return result, nil
	}
	result.ParsedPlan = parsedPlan

	// Save the plan content
	if err := p.planManager.SavePlan(planInfo, parsedPlan.RawContent); err != nil {
		result.Error = fmt.Errorf("failed to save plan: %w", err)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// IteratePlan iterates on an existing plan with feedback
func (p *Planner) IteratePlan(ctx context.Context, planInfo *plan.PlanInfo, feedback string) (*PlanResult, error) {
	result := &PlanResult{PlanInfo: planInfo}

	// Load existing plan
	existingPlan, err := p.planManager.LoadPlan(planInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing plan: %w", err)
	}

	// Build iteration prompt
	prompt := p.buildIterationPrompt(existingPlan, feedback)

	// Execute Claude Code
	execResult, err := p.executor.Execute(ctx, prompt, plannerSystemPrompt)
	if err != nil {
		result.Error = err
		return result, nil
	}

	result.RawOutput = execResult.Output
	result.Tokens = execResult.Tokens

	if !execResult.Success {
		result.Error = execResult.Error
		return result, nil
	}

	// Parse the updated plan
	parsedPlan, err := ParsePlanFromOutput(execResult.Output)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse plan: %w", err)
		return result, nil
	}
	result.ParsedPlan = parsedPlan

	// Save the updated plan
	if err := p.planManager.SavePlan(planInfo, parsedPlan.RawContent); err != nil {
		result.Error = fmt.Errorf("failed to save plan: %w", err)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// IteratePlanWithStreaming iterates on an existing plan with streaming output
func (p *Planner) IteratePlanWithStreaming(ctx context.Context, planInfo *plan.PlanInfo, feedback string, callback func(line string)) (*PlanResult, error) {
	result := &PlanResult{PlanInfo: planInfo}

	// Load existing plan
	existingPlan, err := p.planManager.LoadPlan(planInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing plan: %w", err)
	}

	// Build iteration prompt
	prompt := p.buildIterationPrompt(existingPlan, feedback)

	// Execute Claude Code with streaming
	execResult, err := p.executor.ExecuteWithStreaming(ctx, prompt, plannerSystemPrompt, callback)
	if err != nil {
		result.Error = err
		return result, nil
	}

	result.RawOutput = execResult.Output
	result.Tokens = execResult.Tokens

	if !execResult.Success {
		result.Error = execResult.Error
		return result, nil
	}

	// Parse the updated plan
	parsedPlan, err := ParsePlanFromOutput(execResult.Output)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse plan: %w", err)
		return result, nil
	}
	result.ParsedPlan = parsedPlan

	// Save the updated plan
	if err := p.planManager.SavePlan(planInfo, parsedPlan.RawContent); err != nil {
		result.Error = fmt.Errorf("failed to save plan: %w", err)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// buildPlanPrompt builds the prompt for plan generation
func (p *Planner) buildPlanPrompt(task string) string {
	return fmt.Sprintf(`Create a detailed implementation plan for the following task:

%s

Please provide a structured plan with the following sections:
1. Overview - Brief description of the approach
2. Steps - Numbered list of implementation steps
3. Files to Create - List of new files to create with paths
4. Files to Modify - List of existing files to modify
5. Dependencies - Any packages or dependencies needed

Wrap your plan between # PLAN_START and # PLAN_END markers.`, task)
}

// buildIterationPrompt builds the prompt for plan iteration
func (p *Planner) buildIterationPrompt(existingPlan, feedback string) string {
	return fmt.Sprintf(`Here is the current implementation plan:

%s

User feedback:
%s

Please update the plan based on this feedback. Keep the same structure with Overview, Steps, Files to Create, Files to Modify, and Dependencies sections.

Wrap your updated plan between # PLAN_START and # PLAN_END markers.`, existingPlan, feedback)
}

// plannerSystemPrompt is the system prompt for the planner agent
const plannerSystemPrompt = `You are a senior software architect creating implementation plans.
Generate detailed, actionable coding plans with:
- Clear overview of the approach
- Step-by-step implementation guide
- Files to create/modify with exact paths
- Dependencies needed with versions
- Potential risks and mitigations

Be specific about file paths and code changes. Output in markdown format.
Always wrap your plan between # PLAN_START and # PLAN_END markers.`

// FormatPlanForDisplay formats a parsed plan for TUI display
func FormatPlanForDisplay(p *ParsedPlan) string {
	var sb strings.Builder

	if p.Title != "" {
		sb.WriteString("# ")
		sb.WriteString(p.Title)
		sb.WriteString("\n\n")
	}

	if p.Overview != "" {
		sb.WriteString("## Overview\n")
		sb.WriteString(p.Overview)
		sb.WriteString("\n\n")
	}

	if len(p.Steps) > 0 {
		sb.WriteString("## Steps\n")
		for i, step := range p.Steps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
		sb.WriteString("\n")
	}

	if len(p.FilesToCreate) > 0 {
		sb.WriteString("## Files to Create\n")
		for _, file := range p.FilesToCreate {
			sb.WriteString(fmt.Sprintf("- `%s`\n", file))
		}
		sb.WriteString("\n")
	}

	if len(p.FilesToModify) > 0 {
		sb.WriteString("## Files to Modify\n")
		for _, file := range p.FilesToModify {
			sb.WriteString(fmt.Sprintf("- `%s`\n", file))
		}
		sb.WriteString("\n")
	}

	if len(p.Dependencies) > 0 {
		sb.WriteString("## Dependencies\n")
		for _, dep := range p.Dependencies {
			sb.WriteString(fmt.Sprintf("- %s\n", dep))
		}
	}

	return sb.String()
}

