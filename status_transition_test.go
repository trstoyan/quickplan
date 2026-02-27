package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTransitionTestManager(t *testing.T) (*ProjectDataManager, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "quickplan-transition-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.3.0-alpha.rc1"))
	projectName := "transition-project"
	if err := pdm.CreateProject(projectName); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create project: %v", err)
	}

	return pdm, projectName, func() { os.RemoveAll(tmpDir) }
}

func TestUpdateTaskStatus_LegacyTransitionFlow(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	projectData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	projectData.Tasks = []Task{
		{ID: 1, Text: "task", Status: "TODO", Created: time.Now()},
	}
	if err := pdm.SaveProjectData(projectName, projectData); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if err := pdm.UpdateTaskStatus(projectName, "t-1", "IN_PROGRESS", "agent-1"); err != nil {
		t.Fatalf("expected IN_PROGRESS transition to pass: %v", err)
	}
	if err := pdm.UpdateTaskStatus(projectName, "t-1", "DONE", "agent-1"); err != nil {
		t.Fatalf("expected DONE transition to pass: %v", err)
	}

	reloaded, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if got := GetTaskStatus(reloaded.Tasks[0]); got != "DONE" {
		t.Fatalf("expected DONE, got %s", got)
	}
}

func TestUpdateTaskStatus_LegacyInvalidTransition(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	projectData, _ := pdm.LoadProjectData(projectName)
	projectData.Tasks = []Task{
		{ID: 1, Text: "task", Status: "TODO", Created: time.Now()},
	}
	_ = pdm.SaveProjectData(projectName, projectData)

	err := pdm.UpdateTaskStatus(projectName, "t-1", "DONE", "agent-1")
	if err == nil {
		t.Fatal("expected invalid transition error")
	}
	if !strings.Contains(err.Error(), "invalid transition") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTaskStatus_DependencyAndGuardChecks(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	projectData, _ := pdm.LoadProjectData(projectName)
	projectData.Tasks = []Task{
		{ID: 1, Text: "dep", Status: "TODO", Created: time.Now()},
		{ID: 2, Text: "main", Status: "TODO", DependsOn: []int{1}, WatchPath: filepath.Join(os.TempDir(), "quickplan-missing-guard"), Created: time.Now()},
	}
	_ = pdm.SaveProjectData(projectName, projectData)

	err := pdm.UpdateTaskStatus(projectName, "t-2", "IN_PROGRESS", "agent-2")
	if err == nil || !strings.Contains(err.Error(), "dependency") {
		t.Fatalf("expected dependency readiness error, got: %v", err)
	}

	projectData, _ = pdm.LoadProjectData(projectName)
	projectData.Tasks[0].Status = "DONE"
	projectData.Tasks[0].Done = true
	_ = pdm.SaveProjectData(projectName, projectData)

	err = pdm.UpdateTaskStatus(projectName, "t-2", "IN_PROGRESS", "agent-2")
	if err == nil || !strings.Contains(err.Error(), "watch path") {
		t.Fatalf("expected watch path readiness error, got: %v", err)
	}

	guardFile := filepath.Join(os.TempDir(), "quickplan-existing-guard")
	if writeErr := os.WriteFile(guardFile, []byte("ok"), 0644); writeErr != nil {
		t.Fatalf("failed to create guard file: %v", writeErr)
	}
	defer os.Remove(guardFile)

	projectData, _ = pdm.LoadProjectData(projectName)
	projectData.Tasks[1].WatchPath = guardFile
	_ = pdm.SaveProjectData(projectName, projectData)

	if err := pdm.UpdateTaskStatus(projectName, "t-2", "IN_PROGRESS", "agent-2"); err != nil {
		t.Fatalf("expected guarded transition to pass: %v", err)
	}
}

func TestUpdateTaskStatus_V11RetryLoop(t *testing.T) {
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
			{ID: "t-1", Name: "task", Status: "TODO", UpdatedAt: time.Now()},
		},
		Events: []Event{},
	}
	if err := pdm.SaveProjectV11(projectName, v11); err != nil {
		t.Fatalf("save v1.1 failed: %v", err)
	}

	if err := pdm.UpdateTaskStatus(projectName, "t-1", "IN_PROGRESS", "agent-1"); err != nil {
		t.Fatalf("TODO -> IN_PROGRESS failed: %v", err)
	}
	if err := pdm.UpdateTaskStatus(projectName, "t-1", "FAILED", "agent-1"); err != nil {
		t.Fatalf("IN_PROGRESS -> FAILED failed: %v", err)
	}
	if err := pdm.UpdateTaskStatus(projectName, "t-1", "RETRYING", "agent-1"); err != nil {
		t.Fatalf("FAILED -> RETRYING failed: %v", err)
	}
	if err := pdm.UpdateTaskStatus(projectName, "t-1", "PENDING", "agent-1"); err != nil {
		t.Fatalf("RETRYING -> PENDING failed: %v", err)
	}

	err := pdm.UpdateTaskStatus(projectName, "t-1", "DONE", "agent-1")
	if err == nil || !strings.Contains(err.Error(), "invalid transition") {
		t.Fatalf("expected invalid transition error from PENDING -> DONE, got: %v", err)
	}
}

func TestReconcileTaskReadiness_LegacyBlockAndUnblock(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	guardPath := filepath.Join(t.TempDir(), "guard.txt")

	projectData, _ := pdm.LoadProjectData(projectName)
	projectData.Tasks = []Task{
		{ID: 1, Text: "guarded task", Status: "TODO", WatchPath: guardPath, Created: time.Now()},
	}
	_ = pdm.SaveProjectData(projectName, projectData)

	changed, err := pdm.ReconcileTaskReadiness(projectName, "daemon")
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	if changed != 1 {
		t.Fatalf("expected 1 changed task, got %d", changed)
	}

	reloaded, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if got := GetTaskStatus(reloaded.Tasks[0]); got != "BLOCKED" {
		t.Fatalf("expected BLOCKED after failed guard, got %s", got)
	}

	if err := os.WriteFile(guardPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create guard path: %v", err)
	}

	changed, err = pdm.ReconcileTaskReadiness(projectName, "daemon")
	if err != nil {
		t.Fatalf("second reconcile failed: %v", err)
	}
	if changed != 1 {
		t.Fatalf("expected 1 changed task on unblock, got %d", changed)
	}

	reloaded, err = pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if got := GetTaskStatus(reloaded.Tasks[0]); got != "PENDING" {
		t.Fatalf("expected PENDING after guard recovery, got %s", got)
	}

	eventLog, err := pdm.LoadEvents(projectName)
	if err != nil {
		t.Fatalf("load events failed: %v", err)
	}
	hasBlockedEvent := false
	hasUnblockedEvent := false
	for _, e := range eventLog.Events {
		if e.Type == "TASK_BLOCKED" {
			hasBlockedEvent = true
		}
		if e.Type == "TASK_UNBLOCKED" {
			hasUnblockedEvent = true
		}
	}
	if !hasBlockedEvent || !hasUnblockedEvent {
		t.Fatalf("expected TASK_BLOCKED and TASK_UNBLOCKED events, got %+v", eventLog.Events)
	}
}

func TestReconcileTaskReadiness_V11DependencyBlockAndUnblock(t *testing.T) {
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
			{ID: "t-1", Name: "dep", Status: "TODO", UpdatedAt: time.Now()},
			{ID: "t-2", Name: "main", Status: "TODO", DependsOn: []string{"t-1"}, UpdatedAt: time.Now()},
		},
		Events: []Event{},
	}
	if err := pdm.SaveProjectV11(projectName, v11); err != nil {
		t.Fatalf("save v1.1 failed: %v", err)
	}

	changed, err := pdm.ReconcileTaskReadiness(projectName, "daemon")
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	if changed != 1 {
		t.Fatalf("expected one blocked task, got %d", changed)
	}

	reloaded, err := pdm.LoadProjectV11(projectName)
	if err != nil {
		t.Fatalf("reload v1.1 failed: %v", err)
	}
	if reloaded.Tasks[1].Status != "BLOCKED" {
		t.Fatalf("expected t-2 BLOCKED, got %s", reloaded.Tasks[1].Status)
	}

	if err := pdm.UpdateTaskStatus(projectName, "t-1", "IN_PROGRESS", "agent-1"); err != nil {
		t.Fatalf("t-1 IN_PROGRESS failed: %v", err)
	}
	if err := pdm.UpdateTaskStatus(projectName, "t-1", "DONE", "agent-1"); err != nil {
		t.Fatalf("t-1 DONE failed: %v", err)
	}

	changed, err = pdm.ReconcileTaskReadiness(projectName, "daemon")
	if err != nil {
		t.Fatalf("second reconcile failed: %v", err)
	}
	if changed != 1 {
		t.Fatalf("expected one unblocked task, got %d", changed)
	}

	reloaded, err = pdm.LoadProjectV11(projectName)
	if err != nil {
		t.Fatalf("reload v1.1 failed: %v", err)
	}
	if reloaded.Tasks[1].Status != "PENDING" {
		t.Fatalf("expected t-2 PENDING, got %s", reloaded.Tasks[1].Status)
	}
}
