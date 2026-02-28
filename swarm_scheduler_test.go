package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaimNextRunnableTask_RespectsReadiness(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	projectData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	projectData.Tasks = []Task{
		{ID: 1, Text: "dep", Status: "TODO", Created: time.Now()},
		{ID: 2, Text: "main", Status: "TODO", DependsOn: []int{1}, Created: time.Now()},
	}
	if err := pdm.SaveProjectData(projectName, projectData); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	first, err := pdm.ClaimNextRunnableTask(projectName, "worker-1")
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	if first == nil || first.ID != "t-1" {
		t.Fatalf("expected t-1 to be claimed first, got %+v", first)
	}

	second, err := pdm.ClaimNextRunnableTask(projectName, "worker-2")
	if err != nil {
		t.Fatalf("second claim failed: %v", err)
	}
	if second != nil {
		t.Fatalf("expected no second claim while dependency unresolved, got %+v", second)
	}

	if err := pdm.UpdateTaskStatus(projectName, "t-1", "DONE", "worker-1"); err != nil {
		t.Fatalf("failed to complete dependency: %v", err)
	}

	third, err := pdm.ClaimNextRunnableTask(projectName, "worker-2")
	if err != nil {
		t.Fatalf("third claim failed: %v", err)
	}
	if third == nil || third.ID != "t-2" {
		t.Fatalf("expected t-2 after dependency completion, got %+v", third)
	}
}

func TestGetExecutionSnapshot_ReportsTerminalState(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	v11 := &ProjectV11{
		SchemaVersion: "1.1",
		Project: ProjectMeta{
			Name:      projectName,
			Version:   "0.3.0-alpha.rc1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Lock: LockConfig{
			File:       ".quickplan.lock",
			TTLSeconds: 300,
		},
		Tasks: []TaskV11{
			{ID: "t-1", Name: "done", Status: "DONE", UpdatedAt: time.Now()},
			{ID: "t-2", Name: "failed", Status: "FAILED", UpdatedAt: time.Now()},
			{ID: "t-3", Name: "cancelled", Status: "CANCELLED", UpdatedAt: time.Now()},
		},
		Events: []Event{},
	}
	if err := pdm.SaveProjectV11(projectName, v11); err != nil {
		t.Fatalf("save v1.1 failed: %v", err)
	}

	snapshot, err := pdm.GetExecutionSnapshot(projectName)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if !snapshot.AllTerminal {
		t.Fatalf("expected terminal snapshot, got: %s", snapshot.Summary())
	}
	if snapshot.Done != 1 || snapshot.Failed != 1 || snapshot.Cancelled != 1 {
		t.Fatalf("unexpected terminal counters: %s", snapshot.Summary())
	}
}

func TestRunSwarmToCompletion_StallsOnBlockedProject(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	missing := filepath.Join(t.TempDir(), "missing-guard-file")
	projectData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	projectData.Tasks = []Task{
		{ID: 1, Text: "guarded", Status: "TODO", WatchPath: missing, Created: time.Now()},
	}
	if err := pdm.SaveProjectData(projectName, projectData); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	runner := &BackgroundRunner{ProjectManager: pdm}
	err = runSwarmToCompletion(projectName, 1, 25*time.Millisecond, 250*time.Millisecond, runner, pdm, nil)
	if err == nil {
		t.Fatal("expected stalled swarm error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "stalled") {
		t.Fatalf("expected stalled error, got: %v", err)
	}
}

func TestRunSwarmToCompletion_CompletesSingleTask(t *testing.T) {
	t.Setenv("QUICKPLAN_DISABLE_LOCAL_SANDBOX", "1")

	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	projectData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	projectData.Tasks = []Task{
		{
			ID:      1,
			Text:    "ship it",
			Status:  "TODO",
			Created: time.Now(),
			Behavior: AgentBehavior{
				LifeCycle: "Atomic",
				Command:   "echo done",
			},
		},
	}
	if err := pdm.SaveProjectData(projectName, projectData); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	runner := &BackgroundRunner{ProjectManager: pdm}
	if err := runSwarmToCompletion(projectName, 1, 25*time.Millisecond, 5*time.Second, runner, pdm, nil); err != nil {
		t.Fatalf("unexpected swarm completion error: %v", err)
	}

	views, _, err := pdm.GetTaskViews(projectName)
	if err != nil {
		t.Fatalf("failed to load task views: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected one task, got %d", len(views))
	}
	if got := views[0].Status; got != "DONE" {
		t.Fatalf("expected DONE, got %s", got)
	}
}
