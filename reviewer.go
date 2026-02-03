package claude

import (
	"context"
	"fmt"
	"strings"

	"github.com/ravi/agentflow/internal/config"
	"github.com/ravi/agentflow/internal/plan"
	"github.com/ravi/agentflow/internal/process"
)

// Reviewer handles staff engineer review with Claude Code
type Reviewer struct {
	executor    *Executor
	planManager *plan.Manager
	config      *config.Config
}

// ReviewResult holds the result of a review
type ReviewResult struct {
	PlanInfo     *plan.PlanInfo
	ParsedReview *ParsedReview
	RawOutput    string
	Tokens       int
	Success      bool
	Error        error
}

// NewReviewer creates a new reviewer
func NewReviewer(cfg *config.Config, pm *process.Manager, planMgr *plan.Manager) *Reviewer {
	return &Reviewer{
		executor:    NewExecutor(cfg, pm),
		planManager: planMgr,
		config:      cfg,
	}
}

// Review performs a staff engineer review on a plan
func (r *Reviewer) Review(ctx context.Context, planInfo *plan.PlanInfo) (*ReviewResult, error) {
	result := &ReviewResult{PlanInfo: planInfo}

	// Load the plan
	planContent, err := r.planManager.LoadPlan(planInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}

	// Build the review prompt
	prompt := r.buildReviewPrompt(planContent)

	// Execute Claude Code with reviewer system prompt
	execResult, err := r.executor.Execute(ctx, prompt, reviewerSystemPrompt)
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

	// Parse the review
	parsedReview, err := ParseReviewFromOutput(execResult.Output)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse review: %w", err)
		return result, nil
	}
	result.ParsedReview = parsedReview

	// Save the reviewed plan
	if err := r.planManager.SaveReview(planInfo, parsedReview.RawContent); err != nil {
		result.Error = fmt.Errorf("failed to save review: %w", err)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// ReviewWithStreaming performs a review with streaming output
func (r *Reviewer) ReviewWithStreaming(ctx context.Context, planInfo *plan.PlanInfo, callback func(line string)) (*ReviewResult, error) {
	result := &ReviewResult{PlanInfo: planInfo}

	// Load the plan
	planContent, err := r.planManager.LoadPlan(planInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}

	// Build the review prompt
	prompt := r.buildReviewPrompt(planContent)

	// Execute Claude Code with streaming
	execResult, err := r.executor.ExecuteWithStreaming(ctx, prompt, reviewerSystemPrompt, callback)
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

	// Parse the review
	parsedReview, err := ParseReviewFromOutput(execResult.Output)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse review: %w", err)
		return result, nil
	}
	result.ParsedReview = parsedReview

	// Save the reviewed plan
	if err := r.planManager.SaveReview(planInfo, parsedReview.RawContent); err != nil {
		result.Error = fmt.Errorf("failed to save review: %w", err)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// IterateReview iterates on a review with feedback
func (r *Reviewer) IterateReview(ctx context.Context, planInfo *plan.PlanInfo, feedback string) (*ReviewResult, error) {
	result := &ReviewResult{PlanInfo: planInfo}

	// Load existing plan and review
	planContent, err := r.planManager.LoadPlan(planInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}

	reviewContent, err := r.planManager.LoadReview(planInfo)
	if err != nil {
		// If no review exists, perform initial review
		return r.Review(ctx, planInfo)
	}

	// Build iteration prompt
	prompt := r.buildIterationPrompt(planContent, reviewContent, feedback)

	// Execute Claude Code
	execResult, err := r.executor.Execute(ctx, prompt, reviewerSystemPrompt)
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

	// Parse the updated review
	parsedReview, err := ParseReviewFromOutput(execResult.Output)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse review: %w", err)
		return result, nil
	}
	result.ParsedReview = parsedReview

	// Save the updated review
	if err := r.planManager.SaveReview(planInfo, parsedReview.RawContent); err != nil {
		result.Error = fmt.Errorf("failed to save review: %w", err)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// IterateReviewWithStreaming iterates on a review with streaming output
func (r *Reviewer) IterateReviewWithStreaming(ctx context.Context, planInfo *plan.PlanInfo, feedback string, callback func(line string)) (*ReviewResult, error) {
	result := &ReviewResult{PlanInfo: planInfo}

	// Load existing plan and review
	planContent, err := r.planManager.LoadPlan(planInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}

	reviewContent, err := r.planManager.LoadReview(planInfo)
	if err != nil {
		// If no review exists, perform initial review with streaming
		return r.ReviewWithStreaming(ctx, planInfo, callback)
	}

	// Build iteration prompt
	prompt := r.buildIterationPrompt(planContent, reviewContent, feedback)

	// Execute Claude Code with streaming
	execResult, err := r.executor.ExecuteWithStreaming(ctx, prompt, reviewerSystemPrompt, callback)
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

	// Parse the updated review
	parsedReview, err := ParseReviewFromOutput(execResult.Output)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse review: %w", err)
		return result, nil
	}
	result.ParsedReview = parsedReview

	// Save the updated review
	if err := r.planManager.SaveReview(planInfo, parsedReview.RawContent); err != nil {
		result.Error = fmt.Errorf("failed to save review: %w", err)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// buildReviewPrompt builds the prompt for plan review
func (r *Reviewer) buildReviewPrompt(planContent string) string {
	return fmt.Sprintf(`Please review the following implementation plan as a staff engineer with 10+ years of experience:

%s

Provide a thorough review covering:
1. Review Summary - Overall assessment
2. Findings - List items with Good:, Concern:, or Warning: prefixes
3. Approved Steps (with modifications) - The final approved steps

Be constructive but thorough. Identify security issues, edge cases, scalability concerns, and best practices.

Wrap your review between # REVIEW_START and # REVIEW_END markers.`, planContent)
}

// buildIterationPrompt builds the prompt for review iteration
func (r *Reviewer) buildIterationPrompt(planContent, reviewContent, feedback string) string {
	return fmt.Sprintf(`Here is the implementation plan:

%s

Here is the current review:

%s

User feedback:
%s

Please update the review based on this feedback. Keep the same structure with Review Summary, Findings, and Approved Steps sections.

Wrap your updated review between # REVIEW_START and # REVIEW_END markers.`, planContent, reviewContent, feedback)
}

// reviewerSystemPrompt is the system prompt for the reviewer agent
const reviewerSystemPrompt = `You are a staff engineer with 10+ years of experience reviewing code plans.
Your role is to critically review implementation plans and:
- Identify missing edge cases
- Suggest security improvements
- Point out scalability concerns
- Recommend best practices
- Approve or request changes

Be thorough but constructive. Use the following format for findings:
- Good: [Positive observations]
- Concern: [Issues to address]
- Warning: [Critical issues]

Always output the final approved plan with your modifications.
Always wrap your review between # REVIEW_START and # REVIEW_END markers.`

// FormatReviewForDisplay formats a parsed review for TUI display
func FormatReviewForDisplay(r *ParsedReview) string {
	var sb strings.Builder

	sb.WriteString("# Staff Engineer Review\n\n")

	if r.Summary != "" {
		sb.WriteString("## Review Summary\n")
		sb.WriteString(r.Summary)
		sb.WriteString("\n\n")
	}

	if len(r.Findings) > 0 {
		sb.WriteString("## Findings\n")
		for _, finding := range r.Findings {
			prefix := ""
			switch finding.Type {
			case "good":
				prefix = "✓"
			case "concern":
				prefix = "⚠"
			case "warning":
				prefix = "✗"
			default:
				prefix = "•"
			}
			sb.WriteString(fmt.Sprintf("%s %s\n", prefix, finding.Content))
		}
		sb.WriteString("\n")
	}

	if len(r.ApprovedSteps) > 0 {
		sb.WriteString("## Approved Steps\n")
		for i, step := range r.ApprovedSteps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	}

	return sb.String()
}

