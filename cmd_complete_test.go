package main

import (
	"strings"
	"testing"
	"time"
)

func TestCompleteTaskWithTransitions_FromTodoToDone(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	projectData, _ := pdm.LoadProjectData(projectName)
	projectData.Tasks = []Task{
		{ID: 1, Text: "task", Status: "TODO", Created: time.Now()},
	}
	_ = pdm.SaveProjectData(projectName, projectData)

	if err := completeTaskWithTransitions(pdm, projectName, "t-1", "TODO", "human"); err != nil {
		t.Fatalf("expected completion to succeed, got: %v", err)
	}

	reloaded, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if got := GetTaskStatus(reloaded.Tasks[0]); got != "DONE" {
		t.Fatalf("expected DONE, got %s", got)
	}
}

func TestCompleteTaskWithTransitions_BlockedFails(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	projectData, _ := pdm.LoadProjectData(projectName)
	projectData.Tasks = []Task{
		{ID: 1, Text: "task", Status: "BLOCKED", Created: time.Now()},
	}
	_ = pdm.SaveProjectData(projectName, projectData)

	err := completeTaskWithTransitions(pdm, projectName, "t-1", "BLOCKED", "human")
	if err == nil {
		t.Fatal("expected BLOCKED error")
	}
	if !strings.Contains(err.Error(), "BLOCKED") {
		t.Fatalf("unexpected error: %v", err)
	}
}
