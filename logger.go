package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger handles audit logging in JSONL format
type Logger struct {
	filePath string
	mu       sync.Mutex
	file     *os.File
}

// Entry represents an audit log entry
type Entry struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Phase       string    `json:"phase"`
	Action      string    `json:"action"`
	Status      string    `json:"status,omitempty"`
	Model       string    `json:"model,omitempty"`
	InputTokens int       `json:"input_tokens,omitempty"`
	OutputTokens int      `json:"output_tokens,omitempty"`
	DurationMs  int64     `json:"duration_ms,omitempty"`
	CostUSD     float64   `json:"cost_usd,omitempty"`
	UserAction  string    `json:"user_action,omitempty"`
	Error       string    `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Summary holds aggregated audit information
type Summary struct {
	TotalEntries   int
	TotalRequests  int
	TotalTokens    int
	InputTokens    int
	OutputTokens   int
	TotalCost      float64
	TotalDuration  time.Duration
	Retries        int
	ApprovalsPassed int
	ApprovalsTotal int
	Errors         int
}

// NewLogger creates a new audit logger
func NewLogger(planDir string) *Logger {
	return &Logger{
		filePath: filepath.Join(planDir, "audit.jsonl"),
	}
}

// Open opens the audit log file for writing
func (l *Logger) Open() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(l.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create audit directory: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit file: %w", err)
	}

	l.file = file
	return nil
}

// Close closes the audit log file
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// Log writes an entry to the audit log
func (l *Logger) Log(entry Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Auto-open if not open
	if l.file == nil {
		if err := l.openLocked(); err != nil {
			return err
		}
	}

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = generateEntryID()
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	// Write line
	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	return nil
}

// openLocked opens the file (must be called with lock held)
func (l *Logger) openLocked() error {
	dir := filepath.Dir(l.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create audit directory: %w", err)
	}

	file, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit file: %w", err)
	}

	l.file = file
	return nil
}

// LogLLMCall logs an LLM call
func (l *Logger) LogLLMCall(phase, model string, inputTokens, outputTokens int, duration time.Duration, cost float64, success bool, err error) error {
	entry := Entry{
		Phase:        phase,
		Action:       "llm_call",
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		DurationMs:   duration.Milliseconds(),
		CostUSD:      cost,
	}

	if success {
		entry.Status = "success"
	} else {
		entry.Status = "error"
		if err != nil {
			entry.Error = err.Error()
		}
	}

	return l.Log(entry)
}

// LogApprovalGate logs an approval gate event
func (l *Logger) LogApprovalGate(phase, action string, approved bool) error {
	entry := Entry{
		Phase:      phase,
		Action:     "approval_gate",
		UserAction: action,
	}

	if approved {
		entry.Status = "approved"
	} else {
		entry.Status = "rejected"
	}

	return l.Log(entry)
}

// LogStateChange logs a state change
func (l *Logger) LogStateChange(fromState, toState, event string) error {
	return l.Log(Entry{
		Action: "state_change",
		Status: "success",
		Metadata: map[string]interface{}{
			"from_state": fromState,
			"to_state":   toState,
			"event":      event,
		},
	})
}

// LogError logs an error
func (l *Logger) LogError(phase, action, errorMsg string) error {
	return l.Log(Entry{
		Phase:  phase,
		Action: action,
		Status: "error",
		Error:  errorMsg,
	})
}

// Read reads all entries from the audit log
func (l *Logger) Read() ([]Entry, error) {
	data, err := os.ReadFile(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("failed to read audit file: %w", err)
	}

	var entries []Entry
	for _, line := range splitLines(string(data)) {
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed entries
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetSummary returns a summary of the audit log
func (l *Logger) GetSummary() (*Summary, error) {
	entries, err := l.Read()
	if err != nil {
		return nil, err
	}

	summary := &Summary{
		TotalEntries: len(entries),
	}

	for _, entry := range entries {
		if entry.Action == "llm_call" {
			summary.TotalRequests++
			summary.InputTokens += entry.InputTokens
			summary.OutputTokens += entry.OutputTokens
			summary.TotalTokens += entry.InputTokens + entry.OutputTokens
			summary.TotalCost += entry.CostUSD
			summary.TotalDuration += time.Duration(entry.DurationMs) * time.Millisecond

			if entry.Status == "error" {
				summary.Errors++
			}
		}

		if entry.Action == "approval_gate" {
			summary.ApprovalsTotal++
			if entry.Status == "approved" {
				summary.ApprovalsPassed++
			}
		}
	}

	return summary, nil
}

// generateEntryID generates a unique entry ID
func generateEntryID() string {
	return fmt.Sprintf("entry_%d", time.Now().UnixNano())
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// FormatSummary formats a summary for display
func FormatSummary(s *Summary) string {
	return fmt.Sprintf(`Duration:        %s
Total Requests:  %d
Total Tokens:    %d (in: %d / out: %d)
Total Cost:      $%.4f
Retries:         %d
Approvals:       %d/%d gates passed`,
		s.TotalDuration.Round(time.Second),
		s.TotalRequests,
		s.TotalTokens,
		s.InputTokens,
		s.OutputTokens,
		s.TotalCost,
		s.Retries,
		s.ApprovalsPassed,
		s.ApprovalsTotal,
	)
}

