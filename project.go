package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ProjectDataManager handles project data operations
// Following Single Responsibility Principle - manages only project data I/O
type ProjectDataManager struct {
	dataDir        string
	versionManager *VersionManager
}

// NewProjectDataManager creates a new project data manager
func NewProjectDataManager(dataDir string, versionManager *VersionManager) *ProjectDataManager {
	return &ProjectDataManager{
		dataDir:        dataDir,
		versionManager: versionManager,
	}
}

// LoadProjectData loads project data from disk with version migration
func (pdm *ProjectDataManager) LoadProjectData(projectName string) (*ProjectData, error) {
	projectPath := filepath.Join(pdm.dataDir, projectName)
	tasksFile := filepath.Join(projectPath, "tasks.yaml")

	var projectData ProjectData

	// Read existing file if it exists
	data, err := os.ReadFile(tasksFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty project data
			return &ProjectData{
				Version:  pdm.versionManager.currentVersion,
				Tasks:    []Task{},
				Created:  time.Now(),
				Modified: time.Now(),
				Archived: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, &projectData); err != nil {
		return nil, fmt.Errorf("failed to parse tasks file: %w", err)
	}

	// Validate version compatibility
	if err := pdm.versionManager.ValidateProjectVersion(projectData.Version); err != nil {
		return nil, err
	}

	// Migrate if needed
	if projectData.Version != pdm.versionManager.currentVersion {
		migrated, err := pdm.versionManager.MigrateProjectIfNeeded(projectPath)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate project: %w", err)
		}
		if migrated {
			// Reload after migration
			data, err := os.ReadFile(tasksFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read migrated tasks file: %w", err)
			}
			if err := yaml.Unmarshal(data, &projectData); err != nil {
				return nil, fmt.Errorf("failed to parse migrated tasks file: %w", err)
			}
		}
	}

	return &projectData, nil
}

// SaveProjectData saves project data to disk with version tracking
func (pdm *ProjectDataManager) SaveProjectData(projectName string, projectData *ProjectData) error {
	projectPath := filepath.Join(pdm.dataDir, projectName)
	tasksFile := filepath.Join(projectPath, "tasks.yaml")

	// Ensure version is set
	if projectData.Version == "" {
		projectData.Version = pdm.versionManager.currentVersion
	}

	// Update modified timestamp
	projectData.Modified = time.Now()

	// Marshal to YAML
	data, err := yaml.Marshal(projectData)
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	// Write to file
	if err := os.WriteFile(tasksFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	return nil
}

// CreateProject creates a new project directory and initializes data
func (pdm *ProjectDataManager) CreateProject(projectName string) error {
	projectPath := filepath.Join(pdm.dataDir, projectName)

	// Create project directory
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	now := time.Now()

	// Initialize project configuration
	projectConfig := &ProjectConfig{
		Name:     projectName,
		Created:  now,
		Modified: now,
		SyncSource: SyncSource{
			Type: "local", // Default to local, can be changed later
		},
	}

	// Save project configuration
	if err := pdm.SaveProjectConfig(projectName, projectConfig); err != nil {
		return fmt.Errorf("failed to save project config: %w", err)
	}

	// Initialize project data
	projectData := &ProjectData{
		Version:  pdm.versionManager.currentVersion,
		Tasks:    []Task{},
		Created:  now,
		Modified: now,
		Archived: false,
	}

	// Save initial tasks data
	return pdm.SaveProjectData(projectName, projectData)
}

// LoadProjectConfig loads project configuration from project.yml
func (pdm *ProjectDataManager) LoadProjectConfig(projectName string) (*ProjectConfig, error) {
	projectPath := filepath.Join(pdm.dataDir, projectName)
	configFile := filepath.Join(projectPath, "project.yml")

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist, create default
			return pdm.createDefaultConfig(projectName)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveProjectConfig saves project configuration to project.yml
func (pdm *ProjectDataManager) SaveProjectConfig(projectName string, config *ProjectConfig) error {
	projectPath := filepath.Join(pdm.dataDir, projectName)
	configFile := filepath.Join(projectPath, "project.yml")

	// Update modified timestamp
	config.Modified = time.Now()

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// createDefaultConfig creates a default project configuration
func (pdm *ProjectDataManager) createDefaultConfig(projectName string) (*ProjectConfig, error) {
	config := &ProjectConfig{
		Name:     projectName,
		Created:  time.Now(),
		Modified: time.Now(),
		SyncSource: SyncSource{
			Type: "local",
		},
	}

	if err := pdm.SaveProjectConfig(projectName, config); err != nil {
		return nil, err
	}

	return config, nil
}

