package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMigrationV11(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "quickplan-migrate-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "legacy-project"
	projectPath := filepath.Join(tmpDir, projectName)
	os.MkdirAll(projectPath, 0755)

	// Create legacy files
	legacyData := ProjectData{
		Version: "0.1.0",
		Tasks: []Task{
			{
				ID:   1,
				Text: "Legacy Task",
				Done: true,
			},
		},
	}
	d, _ := yaml.Marshal(legacyData)
	os.WriteFile(filepath.Join(projectPath, "tasks.yaml"), d, 0644)

	// Run migration via internal logic (or exec if needed, but let's test logic)
	// We'll simulate the command run
	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))
	
	// Check before
	if _, err := os.Stat(filepath.Join(projectPath, "project.yaml")); err == nil {
		t.Fatal("project.yaml should not exist before migration")
	}

	// We'll use a simplified version of the logic in cmd_migrate.go for the test
	// because invoking cobra commands in unit tests is boilerplate-heavy
	
	lData, _ := pdm.LoadProjectData(projectName)
	v11 := ProjectV11{
		SchemaVersion: "1.1",
		Project: ProjectMeta{
			Name: projectName,
		},
		Tasks: make([]TaskV11, len(lData.Tasks)),
	}
	for i, t := range lData.Tasks {
		v11.Tasks[i] = TaskV11{
			ID:   "t-1",
			Name: t.Text,
		}
	}
	pdm.SaveProjectV11(projectName, &v11)

	// Check after
	if _, err := os.Stat(filepath.Join(projectPath, "project.yaml")); os.IsNotExist(err) {
		t.Fatal("project.yaml was not created")
	}

	// Verify dual-read picks it up
	views, isV11, err := pdm.GetTaskViews(projectName)
	if err != nil {
		t.Fatal(err)
	}
	if !isV11 {
		t.Fatal("Expected v1.1 recognition")
	}
	if views[0].ID != "t-1" {
		t.Errorf("Expected migrated ID t-1, got %s", views[0].ID)
	}
}
