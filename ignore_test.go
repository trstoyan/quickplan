package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIgnoreFilter_DefaultPatterns(t *testing.T) {
	filter := NewIgnoreFilter()

	tests := []struct {
		name           string
		dirName        string
		shouldBeIgnored bool
	}{
		{"GitDirectory", ".git", true},
		{"CurrentProject", ".current_project", true},
		{"HiddenDirectory", ".hidden", true},
		{"NormalDirectory", "my-project", false},
		{"AnotherNormalDirectory", "work", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.ShouldIgnore(tt.dirName)
			if result != tt.shouldBeIgnored {
				t.Errorf("ShouldIgnore(%s) = %v, want %v", tt.dirName, result, tt.shouldBeIgnored)
			}
		})
	}
}

func TestIgnoreFilter_LoadIgnoreFile(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "quickplan-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .quickplanignore file
	ignoreContent := `# Test ignore file
temp
backup-*
test_*
`
	ignoreFile := filepath.Join(tmpDir, ".quickplanignore")
	if err := os.WriteFile(ignoreFile, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("failed to create ignore file: %v", err)
	}

	// Load ignore patterns
	filter := NewIgnoreFilter()
	if err := filter.LoadIgnoreFile(tmpDir); err != nil {
		t.Fatalf("failed to load ignore file: %v", err)
	}

	tests := []struct {
		name           string
		dirName        string
		shouldBeIgnored bool
	}{
		{"DefaultGit", ".git", true},
		{"CustomTemp", "temp", true},
		{"CustomBackup", "backup-2024", true},
		{"CustomTest", "test_feature", true},
		{"Normal", "my-project", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.ShouldIgnore(tt.dirName)
			if result != tt.shouldBeIgnored {
				t.Errorf("ShouldIgnore(%s) = %v, want %v", tt.dirName, result, tt.shouldBeIgnored)
			}
		})
	}
}

func TestIgnoreFilter_MatchPattern(t *testing.T) {
	filter := NewIgnoreFilter()

	tests := []struct {
		name    string
		dirName string
		pattern string
		matches bool
	}{
		{"ExactMatch", ".git", ".git", true},
		{"WildcardPrefix", "backup-2024", "backup-*", true},
		{"WildcardSuffix", "test_feature", "test_*", true},
		{"NoMatch", "project", "temp", false},
		{"HiddenWildcard", ".hidden", ".*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.matchPattern(tt.dirName, tt.pattern)
			if result != tt.matches {
				t.Errorf("matchPattern(%s, %s) = %v, want %v", tt.dirName, tt.pattern, result, tt.matches)
			}
		})
	}
}

func TestCreateDefaultIgnoreFile(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "quickplan-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create default ignore file
	if err := CreateDefaultIgnoreFile(tmpDir); err != nil {
		t.Fatalf("failed to create default ignore file: %v", err)
	}

	// Verify file exists
	ignoreFile := filepath.Join(tmpDir, ".quickplanignore")
	if _, err := os.Stat(ignoreFile); os.IsNotExist(err) {
		t.Error(".quickplanignore file was not created")
	}

	// Verify file content is not empty
	content, err := os.ReadFile(ignoreFile)
	if err != nil {
		t.Fatalf("failed to read ignore file: %v", err)
	}

	if len(content) == 0 {
		t.Error("ignore file is empty")
	}

	// Test that calling it again doesn't overwrite
	if err := CreateDefaultIgnoreFile(tmpDir); err != nil {
		t.Fatalf("second call to CreateDefaultIgnoreFile failed: %v", err)
	}

	// File should still exist with same content
	newContent, err := os.ReadFile(ignoreFile)
	if err != nil {
		t.Fatalf("failed to read ignore file after second call: %v", err)
	}

	if string(content) != string(newContent) {
		t.Error("ignore file was overwritten on second call")
	}
}
