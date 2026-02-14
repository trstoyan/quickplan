package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// VersionManager handles version compatibility and migrations
// Following Single Responsibility Principle - handles only version-related operations
type VersionManager struct {
	currentVersion string
}

// NewVersionManager creates a new version manager instance
func NewVersionManager(version string) *VersionManager {
	return &VersionManager{
		currentVersion: version,
	}
}

// MigrateProjectIfNeeded checks and migrates project data to current version
// Returns true if migration occurred, false otherwise
func (vm *VersionManager) MigrateProjectIfNeeded(projectPath string) (bool, error) {
	tasksFile := filepath.Join(projectPath, "tasks.yaml")

	// Read existing data
	data, err := os.ReadFile(tasksFile)
	if err != nil {
		return false, fmt.Errorf("failed to read tasks file: %w", err)
	}

	var projectData ProjectData
	if err := yaml.Unmarshal(data, &projectData); err != nil {
		return false, fmt.Errorf("failed to parse tasks file: %w", err)
	}

	// Check if migration is needed
	if projectData.Version == vm.currentVersion {
		return false, nil // Already up to date
	}

	// Perform migration
	if err := vm.migrateFromVersion(projectData.Version, vm.currentVersion, &projectData); err != nil {
		return false, fmt.Errorf("migration failed: %w", err)
	}

	// Update version
	projectData.Version = vm.currentVersion

	// Save migrated data
	migratedData, err := yaml.Marshal(&projectData)
	if err != nil {
		return false, fmt.Errorf("failed to marshal migrated data: %w", err)
	}

	if err := os.WriteFile(tasksFile, migratedData, 0644); err != nil {
		return false, fmt.Errorf("failed to write migrated data: %w", err)
	}

	return true, nil
}

// migrateFromVersion applies version-specific migrations
// Following Open/Closed Principle - easy to extend with new version migrations
func (vm *VersionManager) migrateFromVersion(from, to string, data *ProjectData) error {
	// Handle empty version (legacy projects)
	if from == "" {
		// Legacy projects don't need data migration, just version tagging
		return nil
	}

	// Future migrations can be added here
	// Example:
	// if from == "0.1.0" && to >= "0.2.0" {
	//     // Apply 0.1.0 -> 0.2.0 migration logic
	// }

	return nil
}

// ValidateProjectVersion checks if a project version is compatible
func (vm *VersionManager) ValidateProjectVersion(projectVersion string) error {
	if projectVersion == "" {
		// Empty version means legacy project, compatible
		return nil
	}

	// For now, all versions are compatible
	// Future: implement semver comparison for breaking changes
	// if !isCompatible(projectVersion, vm.currentVersion) {
	//     return fmt.Errorf("project version %s is not compatible with CLI version %s", projectVersion, vm.currentVersion)
	// }

	return nil
}
