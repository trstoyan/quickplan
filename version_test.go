package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestVersionManager_MigrateProjectIfNeeded(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "quickplan-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectPath := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Test case 1: Migrate project without version (legacy)
	t.Run("MigrateLegacyProject", func(t *testing.T) {
		tasksFile := filepath.Join(projectPath, "tasks.yaml")

		// Create legacy project data without version
		legacyData := ProjectData{
			Tasks:    []Task{},
			Created:  time.Now(),
			Modified: time.Now(),
			Archived: false,
		}

		data, err := yaml.Marshal(&legacyData)
		if err != nil {
			t.Fatalf("failed to marshal legacy data: %v", err)
		}

		if err := os.WriteFile(tasksFile, data, 0644); err != nil {
			t.Fatalf("failed to write legacy file: %v", err)
		}

		// Migrate
		vm := NewVersionManager("0.1.0")
		migrated, err := vm.MigrateProjectIfNeeded(projectPath)
		if err != nil {
			t.Fatalf("migration failed: %v", err)
		}

		if !migrated {
			t.Error("expected migration to occur for legacy project")
		}

		// Verify migrated data
		data, err = os.ReadFile(tasksFile)
		if err != nil {
			t.Fatalf("failed to read migrated file: %v", err)
		}

		var migratedData ProjectData
		if err := yaml.Unmarshal(data, &migratedData); err != nil {
			t.Fatalf("failed to parse migrated file: %v", err)
		}

		if migratedData.Version != "0.1.0" {
			t.Errorf("expected version 0.1.0, got %s", migratedData.Version)
		}
	})

	// Test case 2: No migration needed for current version
	t.Run("NoMigrationNeeded", func(t *testing.T) {
		tasksFile := filepath.Join(projectPath, "tasks.yaml")

		// Create current version project data
		currentData := ProjectData{
			Version:  "0.1.0",
			Tasks:    []Task{},
			Created:  time.Now(),
			Modified: time.Now(),
			Archived: false,
		}

		data, err := yaml.Marshal(&currentData)
		if err != nil {
			t.Fatalf("failed to marshal current data: %v", err)
		}

		if err := os.WriteFile(tasksFile, data, 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		// Attempt migration
		vm := NewVersionManager("0.1.0")
		migrated, err := vm.MigrateProjectIfNeeded(projectPath)
		if err != nil {
			t.Fatalf("migration check failed: %v", err)
		}

		if migrated {
			t.Error("expected no migration for current version project")
		}
	})
}

func TestVersionManager_ValidateProjectVersion(t *testing.T) {
	vm := NewVersionManager("0.1.0")

	tests := []struct {
		name           string
		projectVersion string
		expectError    bool
	}{
		{"EmptyVersion", "", false},
		{"SameVersion", "0.1.0", false},
		{"DifferentVersion", "0.2.0", false}, // Currently all versions compatible
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vm.ValidateProjectVersion(tt.projectVersion)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
		})
	}
}
