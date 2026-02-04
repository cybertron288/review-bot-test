package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *Logger
	once     sync.Once
)

// Logger provides debug logging capabilities
type Logger struct {
	file    *os.File
	mu      sync.Mutex
	enabled bool
}

// Init initializes the global logger
func Init(logDir string) error {
	var err error
	once.Do(func() {
		instance = &Logger{enabled: true}

		// Create log directory if it doesn't exist
		if err = os.MkdirAll(logDir, 0755); err != nil {
			return
		}

		// Create log file with timestamp
		logPath := filepath.Join(logDir, fmt.Sprintf("agentflow_%s.log", time.Now().Format("2006-01-02")))
		instance.file, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	})
	return err
}

// Close closes the logger
func Close() {
	if instance != nil && instance.file != nil {
		instance.file.Close()
	}
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	log("DEBUG", format, args...)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	log("INFO", format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	log("WARN", format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	log("ERROR", format, args...)
}

func log(level, format string, args ...interface{}) {
	if instance == nil || !instance.enabled {
		return
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, msg)

	if instance.file != nil {
		instance.file.WriteString(line)
		instance.file.Sync() // Force flush to disk immediately
	}
}

// SetEnabled enables or disables logging
func SetEnabled(enabled bool) {
	if instance != nil {
		instance.enabled = enabled
	}
}

