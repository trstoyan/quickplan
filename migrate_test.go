package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestMigrationRoundtrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "quickplan-roundtrip-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "complex-legacy"
	projectPath := filepath.Join(tmpDir, projectName)
	os.MkdirAll(projectPath, 0755)

	now := time.Now().Round(time.Second)

	// 1. Create complex legacy fixture
	legacyData := ProjectData{
		Version: "0.1.0",
		Created: now.Add(-1 * time.Hour),
		Tasks: []Task{
			{
				ID:      1,
				Text:    "Task Done",
				Done:    true,
				Created: now.Add(-1 * time.Hour),
			},
			{
				ID:      2,
				Text:    "Task Blocked",
				Done:    false,
				Created: now.Add(-45 * time.Minute),
				Notes: []NoteEntry{
					{Text: "This is BLOCKED by something", Timestamp: now.Add(-30 * time.Minute)},
				},
			},
			{
				ID:        3,
				Text:      "Task Pending",
				Done:      false,
				Created:   now.Add(-15 * time.Minute),
				DependsOn: []int{1},
			},
		},
	}
	d, _ := yaml.Marshal(legacyData)
	os.WriteFile(filepath.Join(projectPath, "tasks.yaml"), d, 0644)

	legacyConfig := ProjectConfig{
		Name:        projectName,
		Description: "Legacy project description",
	}
	c, _ := yaml.Marshal(legacyConfig)
	os.WriteFile(filepath.Join(projectPath, "project.yml"), c, 0644)

	// 2. Perform migration logic (simulated)
	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))

	lData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("Failed to load legacy data: %v", err)
	}

	v11 := ProjectV11{
		SchemaVersion: "1.1",
		Project: ProjectMeta{
			Name:      projectName,
			Version:   "0.1.0",
			CreatedAt: lData.Created,
			UpdatedAt: time.Now(),
		},
		Lock: LockConfig{
			File:       ".quickplan.lock",
			TTLSeconds: 300,
		},
		Tasks: make([]TaskV11, len(lData.Tasks)),
	}

	for i, tsk := range lData.Tasks {
		status := GetTaskStatus(tsk)
		deps := make([]string, len(tsk.DependsOn))
		for j, d := range tsk.DependsOn {
			deps[j] = "t-" + strconv.Itoa(d)
		}

		v11.Tasks[i] = TaskV11{
			ID:        "t-" + strconv.Itoa(tsk.ID),
			Name:      tsk.Text,
			Status:    status,
			DependsOn: deps,
			UpdatedAt: time.Now(),
		}
	}

	err = pdm.SaveProjectV11(projectName, &v11)
	if err != nil {
		t.Fatalf("Failed to save v1.1: %v", err)
	}

	// 3. Validate results
	if _, err := os.Stat(filepath.Join(projectPath, "project.yaml")); os.IsNotExist(err) {
		t.Fatal("project.yaml was not created")
	}

	reloaded, isV11, err := pdm.GetTaskViews(projectName)
	if err != nil {
		t.Fatalf("Failed to reload project: %v", err)
	}
	if !isV11 {
		t.Fatal("Expected project to be v1.1")
	}

	// Verify ID mapping and Status mapping
	statusMap := make(map[string]string)
	for _, v := range reloaded {
		statusMap[v.ID] = v.Status
	}

	if statusMap["t-1"] != "DONE" {
		t.Errorf("Expected t-1 to be DONE, got %s", statusMap["t-1"])
	}
	if statusMap["t-2"] != "BLOCKED" {
		t.Errorf("Expected t-2 to be BLOCKED, got %s", statusMap["t-2"])
	}
	if statusMap["t-3"] != "TODO" {
		t.Errorf("Expected t-3 to be TODO, got %s", statusMap["t-3"])
	}

	// Verify dependencies
	var task3 TaskView
	for _, v := range reloaded {
		if v.ID == "t-3" {
			task3 = v
		}
	}
	if len(task3.DependsOn) != 1 || task3.DependsOn[0] != "t-1" {
		t.Errorf("Task t-3 dependencies wrong: %v", task3.DependsOn)
	}

	// 4. Test operational behavior post-migration
	v11Data, _ := pdm.LoadProjectV11(projectName)
	v11Data.Tasks = append(v11Data.Tasks, TaskV11{
		ID:     "t-4",
		Name:   "New v1.1 Task",
		Status: "TODO",
	})
	err = pdm.SaveProjectV11(projectName, v11Data)
	if err != nil {
		t.Fatalf("Failed to update v1.1 project: %v", err)
	}

	// 5. Ensure legacy files remain unchanged
	legacyDataAfter, _ := os.ReadFile(filepath.Join(projectPath, "tasks.yaml"))
	if string(legacyDataAfter) != string(d) {
		t.Error("Legacy tasks.yaml was modified during migration")
	}
}
