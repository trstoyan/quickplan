package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestLoadProjectV11(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "quickplan-v11-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "v11-project"
	projectPath := filepath.Join(tmpDir, projectName)
	os.MkdirAll(projectPath, 0755)

	v11 := ProjectV11{
		SchemaVersion: "1.1",
		Project: ProjectMeta{
			Name:      "Test v1.1",
			CreatedAt: time.Now(),
		},
		Tasks: []TaskV11{
			{
				ID:     "t-1",
				Name:   "Task 1",
				Status: "TODO",
			},
		},
	}
	data, _ := yaml.Marshal(v11)
	os.WriteFile(filepath.Join(projectPath, "project.yaml"), data, 0644)

	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))

	views, isV11, err := pdm.GetTaskViews(projectName)
	if err != nil {
		t.Fatalf("Failed to get task views: %v", err)
	}

	if !isV11 {
		t.Error("Expected project to be recognized as v1.1")
	}

	if len(views) != 1 {
		t.Errorf("Expected 1 task, got %d", len(views))
	}

	if views[0].ID != "t-1" {
		t.Errorf("Expected task ID t-1, got %s", views[0].ID)
	}
}
