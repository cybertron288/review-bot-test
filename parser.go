package claude

import (
	"strings"
)

// Plan markers
const (
	PlanStartMarker   = "# PLAN_START"
	PlanEndMarker     = "# PLAN_END"
	ReviewStartMarker = "# REVIEW_START"
	ReviewEndMarker   = "# REVIEW_END"
)

// ParsedPlan represents a parsed plan
type ParsedPlan struct {
	Title       string
	Overview    string
	Steps       []string
	FilesToCreate []string
	FilesToModify []string
	Dependencies []string
	RawContent  string
}

// ParsedReview represents a parsed review
type ParsedReview struct {
	Summary       string
	Findings      []Finding
	ApprovedSteps []string
	RawContent    string
}

// Finding represents a review finding
type Finding struct {
	Type    string // "good", "concern", "warning"
	Content string
}

// ParsePlanFromOutput parses a plan from Claude Code output
func ParsePlanFromOutput(output string) (*ParsedPlan, error) {
	plan := &ParsedPlan{}

	// Find plan content between markers
	content := extractBetweenMarkers(output, PlanStartMarker, PlanEndMarker)
	if content == "" {
		// If no markers, use entire output
		content = output
	}
	plan.RawContent = content

	// Parse sections
	lines := strings.Split(content, "\n")
	currentSection := ""
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check for section headers
		if strings.HasPrefix(trimmed, "## ") {
			currentSection = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "# PLAN") && !strings.HasPrefix(trimmed, "# REVIEW") {
			plan.Title = strings.TrimPrefix(trimmed, "# ")
			continue
		}

		// Skip empty lines and markers
		if trimmed == "" || strings.HasPrefix(trimmed, "# PLAN") {
			continue
		}

		// Parse based on current section
		switch {
		case strings.Contains(currentSection, "overview"):
			if plan.Overview != "" {
				plan.Overview += "\n"
			}
			plan.Overview += line

		case strings.Contains(currentSection, "step"):
			if strings.HasPrefix(trimmed, "-") || (len(trimmed) > 0 && trimmed[0] >= '0' && trimmed[0] <= '9') {
				step := strings.TrimLeft(trimmed, "- 0123456789.")
				step = strings.TrimSpace(step)
				if step != "" {
					plan.Steps = append(plan.Steps, step)
				}
			}

		case strings.Contains(currentSection, "files to create"):
			if strings.HasPrefix(trimmed, "-") {
				file := extractFileName(trimmed)
				if file != "" {
					plan.FilesToCreate = append(plan.FilesToCreate, file)
				}
			}

		case strings.Contains(currentSection, "files to modify"):
			if strings.HasPrefix(trimmed, "-") {
				file := extractFileName(trimmed)
				if file != "" {
					plan.FilesToModify = append(plan.FilesToModify, file)
				}
			}

		case strings.Contains(currentSection, "dependenc"):
			if strings.HasPrefix(trimmed, "-") {
				dep := strings.TrimPrefix(trimmed, "- ")
				if dep != "" {
					plan.Dependencies = append(plan.Dependencies, dep)
				}
			}
		}
	}

	return plan, nil
}

// ParseReviewFromOutput parses a review from Claude Code output
func ParseReviewFromOutput(output string) (*ParsedReview, error) {
	review := &ParsedReview{}

	// Find review content between markers
	content := extractBetweenMarkers(output, ReviewStartMarker, ReviewEndMarker)
	if content == "" {
		// If no markers, use entire output
		content = output
	}
	review.RawContent = content

	// Parse sections
	lines := strings.Split(content, "\n")
	currentSection := ""
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check for section headers
		if strings.HasPrefix(trimmed, "## ") {
			currentSection = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			continue
		}

		// Skip empty lines and markers
		if trimmed == "" || strings.HasPrefix(trimmed, "# REVIEW") {
			continue
		}

		// Parse based on current section
		switch {
		case strings.Contains(currentSection, "summary"):
			if review.Summary != "" {
				review.Summary += "\n"
			}
			review.Summary += line

		case strings.Contains(currentSection, "finding"):
			if strings.HasPrefix(trimmed, "-") {
				finding := parseFinding(trimmed)
				if finding.Content != "" {
					review.Findings = append(review.Findings, finding)
				}
			}

		case strings.Contains(currentSection, "approved") || strings.Contains(currentSection, "step"):
			if strings.HasPrefix(trimmed, "-") || (len(trimmed) > 0 && trimmed[0] >= '0' && trimmed[0] <= '9') {
				step := strings.TrimLeft(trimmed, "- 0123456789.")
				step = strings.TrimSpace(step)
				if step != "" {
					review.ApprovedSteps = append(review.ApprovedSteps, step)
				}
			}
		}
	}

	return review, nil
}

// extractBetweenMarkers extracts content between start and end markers
func extractBetweenMarkers(content, startMarker, endMarker string) string {
	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return ""
	}

	// Move past the start marker line
	startIdx = strings.Index(content[startIdx:], "\n")
	if startIdx == -1 {
		return ""
	}
	startIdx += strings.Index(content, startMarker)

	endIdx := strings.Index(content[startIdx:], endMarker)
	if endIdx == -1 {
		return content[startIdx:]
	}

	return content[startIdx : startIdx+endIdx]
}

// extractFileName extracts a file path from a list item
func extractFileName(line string) string {
	// Remove leading dash and spaces
	line = strings.TrimPrefix(line, "- ")
	line = strings.TrimSpace(line)

	// Extract path from backticks if present
	if strings.Contains(line, "`") {
		start := strings.Index(line, "`")
		end := strings.Index(line[start+1:], "`")
		if end > 0 {
			return line[start+1 : start+1+end]
		}
	}

	// Otherwise take the first word
	parts := strings.Fields(line)
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

// parseFinding parses a finding line
func parseFinding(line string) Finding {
	line = strings.TrimPrefix(line, "- ")
	line = strings.TrimSpace(line)

	finding := Finding{Content: line}

	// Determine type based on prefix
	lower := strings.ToLower(line)
	switch {
	case strings.HasPrefix(lower, "good:"):
		finding.Type = "good"
		finding.Content = strings.TrimPrefix(line, "Good:")
		finding.Content = strings.TrimPrefix(finding.Content, "good:")
	case strings.HasPrefix(lower, "concern:"):
		finding.Type = "concern"
		finding.Content = strings.TrimPrefix(line, "Concern:")
		finding.Content = strings.TrimPrefix(finding.Content, "concern:")
	case strings.HasPrefix(lower, "warning:"):
		finding.Type = "warning"
		finding.Content = strings.TrimPrefix(line, "Warning:")
		finding.Content = strings.TrimPrefix(finding.Content, "warning:")
	case strings.HasPrefix(line, "✓") || strings.HasPrefix(line, "✔"):
		finding.Type = "good"
	case strings.HasPrefix(line, "⚠") || strings.HasPrefix(line, "!"):
		finding.Type = "concern"
	case strings.HasPrefix(line, "✗") || strings.HasPrefix(line, "✘"):
		finding.Type = "warning"
	default:
		finding.Type = "info"
	}

	finding.Content = strings.TrimSpace(finding.Content)
	return finding
}

// HasPlanMarkers checks if the output contains plan markers
func HasPlanMarkers(output string) bool {
	return strings.Contains(output, PlanStartMarker) && strings.Contains(output, PlanEndMarker)
}

// HasReviewMarkers checks if the output contains review markers
func HasReviewMarkers(output string) bool {
	return strings.Contains(output, ReviewStartMarker) && strings.Contains(output, ReviewEndMarker)
}

// IsPlanTruncated checks if the plan output appears to be truncated
// (has start marker but no end marker)
func IsPlanTruncated(output string) bool {
	hasStart := strings.Contains(output, PlanStartMarker)
	hasEnd := strings.Contains(output, PlanEndMarker)
	return hasStart && !hasEnd
}

// IsReviewTruncated checks if the review output appears to be truncated
func IsReviewTruncated(output string) bool {
	hasStart := strings.Contains(output, ReviewStartMarker)
	hasEnd := strings.Contains(output, ReviewEndMarker)
	return hasStart && !hasEnd
}

