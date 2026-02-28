package main

import (
	"testing"
	"time"
)

func TestBackgroundRunnerStart_CompletesTask(t *testing.T) {
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
			Text:    "run task",
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

	if err := pdm.UpdateTaskStatus(projectName, "t-1", "IN_PROGRESS", "worker-1"); err != nil {
		t.Fatalf("failed to set IN_PROGRESS: %v", err)
	}

	views, _, err := pdm.GetTaskViews(projectName)
	if err != nil {
		t.Fatalf("failed to load task views: %v", err)
	}

	var task TaskView
	found := false
	for _, v := range views {
		if v.ID == "t-1" {
			task = v
			found = true
			break
		}
	}
	if !found {
		t.Fatal("task t-1 not found")
	}

	runner := &BackgroundRunner{ProjectManager: pdm}
	if err := runner.Start(projectName, "worker-1", &task); err != nil {
		t.Fatalf("runner start failed: %v", err)
	}

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		reloaded, _, err := pdm.GetTaskViews(projectName)
		if err != nil {
			t.Fatalf("reload failed: %v", err)
		}
		for _, v := range reloaded {
			if v.ID == "t-1" && v.Status == "DONE" {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("task did not reach DONE state after background execution")
}
