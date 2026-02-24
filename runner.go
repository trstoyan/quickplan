package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Runner defines the interface for isolated task execution environments
type Runner interface {
	Setup(task *TaskView) error
	Execute(command string, task *TaskView) (string, error)
	Teardown(task *TaskView) error
}

// LocalRunner executes tasks on the local machine using qp-loop.sh
type LocalRunner struct {
	ScriptPath string
	Project    string
	AgentID    string
}

func (r *LocalRunner) Setup(task *TaskView) error {
	// Local setup usually involves ensuring the qp-loop.sh exists (handled by ExtractScripts)
	return nil
}

func (r *LocalRunner) Execute(command string, task *TaskView) (string, error) {
	cmd := exec.Command(filepath.Join(r.ScriptPath, "qp-loop.sh"), r.Project, r.AgentID)
	// For LocalRunner, qp-loop.sh currently handles the full loop. 
	// In v1.2, we might want to pipe the specific command/prompt here.
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return fmt.Sprintf("Started local agent %s (PID: %d)", r.AgentID, cmd.Process.Pid), nil
}

func (r *LocalRunner) Teardown(task *TaskView) error {
	return nil
}

// DaytonaRunner executes tasks in ephemeral sandboxes using Daytona
type DaytonaRunner struct {
	Project string
	AgentID string
}

func (r *DaytonaRunner) Setup(task *TaskView) error {
	// Check if daytona is installed
	if _, err := exec.LookPath("daytona"); err != nil {
		return fmt.Errorf("Daytona provider requested but 'daytona' CLI not found in PATH")
	}

	image := task.Behavior.Environment.Image
	if image == "" {
		image = "default" // Or a sane default for QuickPlan
	}

	workspaceName := fmt.Sprintf("qp-%s-%s", r.Project, r.AgentID)
	fmt.Printf("🏗️ Daytona: Creating workspace %s with image %s...
", workspaceName, image)

	// Example: daytona create --name qp-project-worker-1 --image golang:1.22
	cmd := exec.Command("daytona", "create", "--name", workspaceName, "--image", image)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Daytona workspace creation failed: %v
Output: %s", err, string(output))
	}

	return nil
}

func (r *DaytonaRunner) Execute(command string, task *TaskView) (string, error) {
	workspaceName := fmt.Sprintf("qp-%s-%s", r.Project, r.AgentID)
	fmt.Printf("🚀 Daytona: Executing task %s in workspace %s...
", task.ID, workspaceName)

	// Example: daytona exec qp-project-worker-1 -- "go run main.go"
	cmd := exec.Command("daytona", "exec", workspaceName, "--", "bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("Daytona execution failed: %v", err)
	}

	return string(output), nil
}

func (r *DaytonaRunner) Teardown(task *TaskView) error {
	workspaceName := fmt.Sprintf("qp-%s-%s", r.Project, r.AgentID)
	fmt.Printf("🧹 Daytona: Destroying workspace %s...
", workspaceName)

	cmd := exec.Command("daytona", "delete", workspaceName, "--force")
	return cmd.Run()
}

// GetRunner returns the appropriate runner based on task behavior
func GetRunner(project, agentID, scriptPath string, task *TaskView) Runner {
	provider := task.Behavior.Environment.Provider
	if provider == "daytona" {
		return &DaytonaRunner{
			Project: project,
			AgentID: agentID,
		}
	}

	// Default to local
	return &LocalRunner{
		ScriptPath: scriptPath,
		Project:    project,
		AgentID:    agentID,
	}
}
