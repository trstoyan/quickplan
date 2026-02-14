package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProjectDataManager_CreateProject(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "quickplan-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	vm := NewVersionManager("0.1.0")
	pdm := NewProjectDataManager(tmpDir, vm)

	projectName := "test-project"
	err = pdm.CreateProject(projectName)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Verify project directory exists
	projectPath := filepath.Join(tmpDir, projectName)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("project directory was not created")
	}

	// Verify tasks.yaml exists and has correct version
	projectData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("failed to load created project: %v", err)
	}

	if projectData.Version != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %s", projectData.Version)
	}

	if len(projectData.Tasks) != 0 {
		t.Errorf("expected empty tasks, got %d tasks", len(projectData.Tasks))
	}

	// Verify project.yml exists
	config, err := pdm.LoadProjectConfig(projectName)
	if err != nil {
		t.Fatalf("failed to load project config: %v", err)
	}

	if config.Name != projectName {
		t.Errorf("expected project name %s, got %s", projectName, config.Name)
	}

	if config.SyncSource.Type != "local" {
		t.Errorf("expected sync type local, got %s", config.SyncSource.Type)
	}
}

func TestProjectDataManager_LoadAndSaveProjectData(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "quickplan-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	vm := NewVersionManager("0.1.0")
	pdm := NewProjectDataManager(tmpDir, vm)

	projectName := "test-project"
	err = pdm.CreateProject(projectName)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Load project data
	projectData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("failed to load project data: %v", err)
	}

	// Add a task
	newTask := Task{
		ID:      1,
		Text:    "Test task",
		Done:    false,
		Created: time.Now(),
	}
	projectData.Tasks = append(projectData.Tasks, newTask)

	// Save project data
	err = pdm.SaveProjectData(projectName, projectData)
	if err != nil {
		t.Fatalf("failed to save project data: %v", err)
	}

	// Reload and verify
	reloadedData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("failed to reload project data: %v", err)
	}

	if len(reloadedData.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(reloadedData.Tasks))
	}

	if reloadedData.Tasks[0].Text != "Test task" {
		t.Errorf("expected task text 'Test task', got '%s'", reloadedData.Tasks[0].Text)
	}
}

func TestProjectDataManager_ProjectConfig(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "quickplan-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	vm := NewVersionManager("0.1.0")
	pdm := NewProjectDataManager(tmpDir, vm)

	projectName := "test-project"
	err = pdm.CreateProject(projectName)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Load config
	config, err := pdm.LoadProjectConfig(projectName)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Modify config
	config.Description = "Test project description"
	config.SyncSource.Type = "git"
	config.SyncSource.URL = "git@github.com:test/repo.git"
	config.SyncSource.Branch = "main"

	// Save config
	err = pdm.SaveProjectConfig(projectName, config)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Reload and verify
	reloadedConfig, err := pdm.LoadProjectConfig(projectName)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if reloadedConfig.Description != "Test project description" {
		t.Errorf("expected description to be saved")
	}

	if reloadedConfig.SyncSource.Type != "git" {
		t.Errorf("expected sync type git, got %s", reloadedConfig.SyncSource.Type)
	}

	if reloadedConfig.SyncSource.URL != "git@github.com:test/repo.git" {
		t.Errorf("expected URL to be saved")
	}

	if reloadedConfig.SyncSource.Branch != "main" {
		t.Errorf("expected branch main, got %s", reloadedConfig.SyncSource.Branch)
	}
}
