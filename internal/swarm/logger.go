package swarm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventLogger handles structured logging for the swarm
type EventLogger struct {
	mu         sync.Mutex
	logFile    *os.File
	filePath   string
	OutputJSON bool
}

// Event represents a structured log entry
type Event struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NewEventLogger creates a new logger instance
func NewEventLogger(path string) (*EventLogger, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &EventLogger{
		logFile:  f,
		filePath: path,
	}, nil
}

// Log writes an event to the log file and optionally to stdout
func (l *EventLogger) Log(level, component, message string, data map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	event := Event{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Component: component,
		Message:   message,
		Data:      data,
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		return
	}

	// Write to JSON file
	if l.logFile != nil {
		_, _ = l.logFile.Write(bytes)
		_, _ = l.logFile.Write([]byte("\n"))
	}

	// Also write to stdout/stderr
	if l.OutputJSON {
		fmt.Println(string(bytes))
	} else {
		// We format it simply for the terminal/journal
		fmt.Printf("[%s] %s: %s\n", component, level, message)
	}
}

// Close closes the log file
func (l *EventLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}
