package swarm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/daytonaio/daytona/libs/sdk-go/pkg/daytona"
	"github.com/daytonaio/daytona/libs/sdk-go/pkg/types"
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
	Workspace  string
}

func (r *LocalRunner) Setup(task *TaskView) error {
	taskID := "default"
	if task != nil && task.ID != "" {
		taskID = task.ID
	}

	workspace := filepath.Join("/tmp/quickplan", fmt.Sprintf("task_%s", taskID))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return err
	}
	r.Workspace = workspace
	return nil
}

func (r *LocalRunner) Execute(command string, task *TaskView) (string, error) {
	if r.Workspace == "" {
		if err := r.Setup(task); err != nil {
			return "", err
		}
	}

	cmd := exec.Command(filepath.Join(r.ScriptPath, "qp-loop.sh"), r.Project, r.AgentID)
	cmd.Dir = r.Workspace
	applyLocalSandbox(cmd, r.Workspace)

	// For LocalRunner, qp-loop.sh currently handles the full loop.
	// In v1.2, we might want to pipe the specific command/prompt here.
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return fmt.Sprintf("Started local agent %s (PID: %d)", r.AgentID, cmd.Process.Pid), nil
}

func (r *LocalRunner) Teardown(task *TaskView) error {
	if r.Workspace == "" {
		return nil
	}
	return os.RemoveAll(r.Workspace)
}

// DaytonaRunner executes tasks in ephemeral sandboxes using Daytona
type DaytonaRunner struct {
	Project string
	AgentID string
	Client  *daytona.Client
	Sandbox *daytona.Sandbox
}

func (r *DaytonaRunner) Setup(task *TaskView) error {
	client, err := daytona.NewClient()
	if err != nil {
		return fmt.Errorf("Daytona server unreachable or unauthenticated: %w", err)
	}
	if task == nil {
		return fmt.Errorf("Daytona runner requires a task")
	}

	image := task.Behavior.Environment.Image
	if image == "" {
		image = "default"
	}

	workspaceName := fmt.Sprintf("qp-%s-%s", r.Project, r.AgentID)
	fmt.Printf("🏗️ Daytona: Creating workspace %s with image %s...\n", workspaceName, image)

	params := types.ImageParams{
		SandboxBaseParams: types.SandboxBaseParams{Name: workspaceName},
		Image:             image,
	}

	sandbox, err := client.Create(context.Background(), params)
	if err != nil {
		return fmt.Errorf("Daytona workspace creation failed: %w", err)
	}

	r.Client = client
	r.Sandbox = sandbox
	return nil
}

func (r *DaytonaRunner) Execute(command string, task *TaskView) (string, error) {
	if r.Sandbox == nil {
		if err := r.Setup(task); err != nil {
			return "", err
		}
	}

	workspaceName := fmt.Sprintf("qp-%s-%s", r.Project, r.AgentID)
	if task != nil {
		fmt.Printf("🚀 Daytona: Executing task %s in workspace %s...\n", task.ID, workspaceName)
	} else {
		fmt.Printf("🚀 Daytona: Executing task in workspace %s...\n", workspaceName)
	}

	response, err := r.Sandbox.Process.ExecuteCommand(context.Background(), command)
	if err != nil {
		return "", fmt.Errorf("Daytona execution failed: %w", err)
	}
	if response.ExitCode != 0 {
		return response.Result, fmt.Errorf("Daytona execution failed: exit code %d", response.ExitCode)
	}

	return response.Result, nil
}

func (r *DaytonaRunner) Teardown(task *TaskView) error {
	if r.Sandbox == nil {
		return nil
	}

	workspaceName := fmt.Sprintf("qp-%s-%s", r.Project, r.AgentID)
	fmt.Printf("🧹 Daytona: Destroying workspace %s...\n", workspaceName)

	if err := r.Sandbox.Delete(context.Background()); err != nil {
		return fmt.Errorf("Daytona workspace deletion failed: %w", err)
	}
	r.Sandbox = nil
	r.Client = nil
	return nil
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
