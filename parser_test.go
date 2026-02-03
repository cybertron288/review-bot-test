package claude

import (
	"testing"
)

func TestParsePlanFromOutput(t *testing.T) {
	output := `Some preamble text...

# PLAN_START
# Implementation Plan: Add OAuth2 Authentication

## Overview
Implement OAuth2 authentication using Google provider.

## Steps
1. Install dependencies
2. Create config file
3. Add routes

## Files to Create
- ` + "`src/config/oauth.ts`" + ` - OAuth configuration
- ` + "`src/routes/auth.ts`" + ` - Auth routes

## Files to Modify
- ` + "`src/app.ts`" + ` - Add middleware

## Dependencies
- passport@0.7.0
- passport-google-oauth20@2.0.0
# PLAN_END

Some postamble text...`

	plan, err := ParsePlanFromOutput(output)
	if err != nil {
		t.Fatalf("ParsePlanFromOutput() error = %v", err)
	}

	if plan.Title != "Implementation Plan: Add OAuth2 Authentication" {
		t.Errorf("Title = %q, want %q", plan.Title, "Implementation Plan: Add OAuth2 Authentication")
	}

	if len(plan.Steps) != 3 {
		t.Errorf("Steps count = %d, want 3", len(plan.Steps))
	}

	if len(plan.FilesToCreate) != 2 {
		t.Errorf("FilesToCreate count = %d, want 2", len(plan.FilesToCreate))
	}

	if len(plan.FilesToModify) != 1 {
		t.Errorf("FilesToModify count = %d, want 1", len(plan.FilesToModify))
	}

	if len(plan.Dependencies) != 2 {
		t.Errorf("Dependencies count = %d, want 2", len(plan.Dependencies))
	}
}

func TestParseReviewFromOutput(t *testing.T) {
	output := `# REVIEW_START
# Staff Engineer Review

## Review Summary
The plan looks good overall.

## Findings
- Good: Clear structure
- Concern: Missing error handling
- Warning: No rate limiting

## Approved Steps (with modifications)
1. Install dependencies (approved)
2. Add rate limiting (new)
# REVIEW_END`

	review, err := ParseReviewFromOutput(output)
	if err != nil {
		t.Fatalf("ParseReviewFromOutput() error = %v", err)
	}

	if len(review.Findings) != 3 {
		t.Errorf("Findings count = %d, want 3", len(review.Findings))
	}

	// Check finding types
	types := map[string]int{}
	for _, f := range review.Findings {
		types[f.Type]++
	}

	if types["good"] != 1 {
		t.Errorf("Good findings = %d, want 1", types["good"])
	}
	if types["concern"] != 1 {
		t.Errorf("Concern findings = %d, want 1", types["concern"])
	}
	if types["warning"] != 1 {
		t.Errorf("Warning findings = %d, want 1", types["warning"])
	}

	if len(review.ApprovedSteps) != 2 {
		t.Errorf("ApprovedSteps count = %d, want 2", len(review.ApprovedSteps))
	}
}

func TestHasPlanMarkers(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "has markers",
			output:   "# PLAN_START\nContent\n# PLAN_END",
			expected: true,
		},
		{
			name:     "missing end marker",
			output:   "# PLAN_START\nContent",
			expected: false,
		},
		{
			name:     "missing start marker",
			output:   "Content\n# PLAN_END",
			expected: false,
		},
		{
			name:     "no markers",
			output:   "Just some content",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPlanMarkers(tt.output)
			if result != tt.expected {
				t.Errorf("HasPlanMarkers() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasReviewMarkers(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "has markers",
			output:   "# REVIEW_START\nContent\n# REVIEW_END",
			expected: true,
		},
		{
			name:     "no markers",
			output:   "Just some content",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasReviewMarkers(tt.output)
			if result != tt.expected {
				t.Errorf("HasReviewMarkers() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractFileName(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"- `src/config/oauth.ts` - OAuth configuration", "src/config/oauth.ts"},
		{"- src/routes/auth.ts - Auth routes", "src/routes/auth.ts"},
		{"- ./package.json", "./package.json"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := extractFileName(tt.line)
			if result != tt.expected {
				t.Errorf("extractFileName(%q) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}

func TestParseFinding(t *testing.T) {
	tests := []struct {
		line         string
		expectedType string
	}{
		{"Good: Clear structure", "good"},
		{"good: lowercase", "good"},
		{"Concern: Missing error handling", "concern"},
		{"Warning: No rate limiting", "warning"},
		{"✓ Check mark good", "good"},
		{"⚠ Warning symbol", "concern"},
		{"Regular finding", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := parseFinding("- " + tt.line)
			if result.Type != tt.expectedType {
				t.Errorf("parseFinding(%q).Type = %q, want %q", tt.line, result.Type, tt.expectedType)
			}
		})
	}
}

