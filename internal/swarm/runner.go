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
	SetLogger(logger *EventLogger)
}

// LocalRunner executes tasks on the local machine
type LocalRunner struct {
	Project   string
	AgentID   string
	Workspace string
	Logger    *EventLogger
}

func (r *LocalRunner) SetLogger(logger *EventLogger) {
	r.Logger = logger
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

	if r.Logger != nil {
		r.Logger.Log("INFO", "LocalRunner", fmt.Sprintf("Setup workspace: %s", workspace), map[string]interface{}{
			"project": r.Project,
			"agent":   r.AgentID,
			"task_id": taskID,
		})
	}
	return nil
}

func (r *LocalRunner) Execute(command string, task *TaskView) (string, error) {
	if r.Workspace == "" {
		if err := r.Setup(task); err != nil {
			return "", err
		}
	}

	if r.Logger != nil {
		r.Logger.Log("INFO", "LocalRunner", fmt.Sprintf("Executing command in %s", r.Workspace), map[string]interface{}{
			"command": command,
			"agent":   r.AgentID,
		})
	}

	if command == "" {
		return "", fmt.Errorf("no execution command provided")
	}

	cmd := exec.Command("sh", "-lc", command)
	cmd.Dir = r.Workspace
	applyLocalSandbox(cmd, r.Workspace)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if r.Logger != nil {
			r.Logger.Log("ERROR", "LocalRunner", "Command execution failed", map[string]interface{}{
				"error":  err.Error(),
				"output": string(output),
			})
		}
		return string(output), err
	}
	return string(output), nil
}

func (r *LocalRunner) Teardown(task *TaskView) error {
	if r.Workspace == "" {
		return nil
	}
	if r.Logger != nil {
		r.Logger.Log("INFO", "LocalRunner", "Teardown workspace", map[string]interface{}{
			"workspace": r.Workspace,
		})
	}
	return os.RemoveAll(r.Workspace)
}

// DaytonaRunner executes tasks in ephemeral sandboxes using Daytona
type DaytonaRunner struct {
	Project string
	AgentID string
	Client  *daytona.Client
	Sandbox *daytona.Sandbox
	Logger  *EventLogger
}

func (r *DaytonaRunner) SetLogger(logger *EventLogger) {
	r.Logger = logger
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

	if r.Logger != nil {
		r.Logger.Log("INFO", "DaytonaRunner", fmt.Sprintf("Creating workspace %s", workspaceName), map[string]interface{}{
			"image": image,
		})
	}

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
	msg := fmt.Sprintf("Executing task in workspace %s", workspaceName)
	if task != nil {
		msg = fmt.Sprintf("Executing task %s in workspace %s", task.ID, workspaceName)
	}

	if r.Logger != nil {
		r.Logger.Log("INFO", "DaytonaRunner", msg, nil)
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

	if r.Logger != nil {
		r.Logger.Log("INFO", "DaytonaRunner", fmt.Sprintf("Destroying workspace %s", workspaceName), nil)
	}

	if err := r.Sandbox.Delete(context.Background()); err != nil {
		return fmt.Errorf("Daytona workspace deletion failed: %w", err)
	}
	r.Sandbox = nil
	r.Client = nil
	return nil
}

// GetRunner returns the appropriate runner based on task behavior
func GetRunner(project, agentID string, task *TaskView) Runner {
	provider := task.Behavior.Environment.Provider
	if provider == "daytona" {
		return &DaytonaRunner{
			Project: project,
			AgentID: agentID,
		}
	}

	// Default to local
	return &LocalRunner{
		Project: project,
		AgentID: agentID,
	}
}
