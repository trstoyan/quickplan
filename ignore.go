package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// IgnoreFilter handles filtering of ignored directories
// Following Single Responsibility Principle - handles only ignore logic
type IgnoreFilter struct {
	patterns []string
}

// NewIgnoreFilter creates a new ignore filter with default patterns
func NewIgnoreFilter() *IgnoreFilter {
	return &IgnoreFilter{
		patterns: getDefaultIgnorePatterns(),
	}
}

// getDefaultIgnorePatterns returns the default ignore patterns
func getDefaultIgnorePatterns() []string {
	return []string{
		".git",           // Git repository directory
		".current_project", // QuickPlan internal file (not a project directory)
		".*",             // Hidden files/directories (starts with dot)
		"node_modules",   // Node.js dependencies
		"build",          // Build artifacts
	}
}

// LoadIgnoreFile loads additional ignore patterns from .quickplanignore file
func (f *IgnoreFilter) LoadIgnoreFile(dataDir string) error {
	ignoreFile := filepath.Join(dataDir, ".quickplanignore")

	file, err := os.Open(ignoreFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No ignore file exists, use defaults only
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		f.patterns = append(f.patterns, line)
	}

	return scanner.Err()
}

// ShouldIgnore checks if a directory name should be ignored
func (f *IgnoreFilter) ShouldIgnore(name string) bool {
	for _, pattern := range f.patterns {
		if f.matchPattern(name, pattern) {
			return true
		}
	}
	return false
}

// matchPattern checks if a name matches an ignore pattern
// Supports simple glob patterns
func (f *IgnoreFilter) matchPattern(name, pattern string) bool {
	// Exact match
	if name == pattern {
		return true
	}

	// Wildcard patterns
	if pattern == ".*" && strings.HasPrefix(name, ".") {
		return true
	}

	// Simple glob matching (can be extended)
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		// If pattern is invalid, treat as literal match
		return name == pattern
	}

	return matched
}

// CreateDefaultIgnoreFile creates a .quickplanignore file with defaults
func CreateDefaultIgnoreFile(dataDir string) error {
	ignoreFile := filepath.Join(dataDir, ".quickplanignore")

	// Check if file already exists
	if _, err := os.Stat(ignoreFile); err == nil {
		// File exists, don't overwrite
		return nil
	}

	content := `# QuickPlan Ignore Patterns
# Directories matching these patterns will not be listed as projects
# Patterns support simple glob matching (*, ?, etc.)

# Default patterns (automatically applied even if not listed here)
.git
.*
node_modules
build

# Add your custom patterns below:
# Examples:
# temp
# backup-*
# test_*
`

	return os.WriteFile(ignoreFile, []byte(content), 0644)
}
