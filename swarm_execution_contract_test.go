package main

import (
	"strings"
	"testing"
	"time"
)

func TestResolveTaskExecution_Command(t *testing.T) {
	task := &TaskView{
		ID: "t-1",
		Behavior: AgentBehavior{
			Command: "go test ./...",
		},
	}

	plan, err := resolveTaskExecution(task)
	if err != nil {
		t.Fatalf("expected command plan, got error: %v", err)
	}
	if plan.Command != "go test ./..." || plan.PluginName != "" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestResolveTaskExecution_Plugin(t *testing.T) {
	task := &TaskView{
		ID:         "t-1",
		AssignedTo: "plugin:lint",
	}

	plan, err := resolveTaskExecution(task)
	if err != nil {
		t.Fatalf("expected plugin plan, got error: %v", err)
	}
	if plan.PluginName != "lint" || plan.Command != "" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestResolveTaskExecution_MissingContract(t *testing.T) {
	task := &TaskView{ID: "t-1"}

	if _, err := resolveTaskExecution(task); err == nil {
		t.Fatal("expected missing execution contract error")
	}
}

func TestValidateProjectExecutionContracts_FailsForRunnableWithoutExecutor(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	projectData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	projectData.Tasks = []Task{
		{ID: 1, Text: "no executor", Status: "TODO", Created: time.Now()},
	}
	if err := pdm.SaveProjectData(projectName, projectData); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	err = validateProjectExecutionContracts(pdm, projectName)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "execution contract") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProjectExecutionContracts_IgnoresDoneTask(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	projectData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	now := time.Now()
	projectData.Tasks = []Task{
		{ID: 1, Text: "already done", Done: true, Status: "DONE", Created: now, Completed: &now},
	}
	if err := pdm.SaveProjectData(projectName, projectData); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if err := validateProjectExecutionContracts(pdm, projectName); err != nil {
		t.Fatalf("expected done-only project to pass validation, got %v", err)
	}
}
