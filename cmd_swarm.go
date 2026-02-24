package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// AgentRunner defines the interface for running agents
type AgentRunner interface {
	Start(project, agentID, scriptPath string) error
}

// BackgroundRunner implements AgentRunner using os/exec
type BackgroundRunner struct{}

// Start starts a worker agent using the appropriate runner
func (br *BackgroundRunner) Start(project, agentID, scriptPath string, task *TaskView) error {
	runner := GetRunner(project, agentID, scriptPath, task)
	
	if err := runner.Setup(task); err != nil {
		return fmt.Errorf("runner setup failed: %w", err)
	}

	// For v1.2, we'll start execution. 
	// If it's a long-running agent, Execute might just start the process.
	// For Daytona, we might want to run the handshake command.
	go func() {
		command := fmt.Sprintf("qp-loop.sh %s %s", project, agentID)
		output, err := runner.Execute(command, task)
		if err != nil {
			fmt.Printf("❌ Runner execution failed for %s: %v\nOutput: %s\n", agentID, err, output)
		}
		
		// In a real swarm, we'd wait for completion before teardown
		// For now, we teardown if it's an atomic lifecycle
		if task.Behavior.LifeCycle == "Atomic" {
			runner.Teardown(task)
		}
	}()

	return nil
}

var swarmCmd = &cobra.Command{
	Use:   "swarm",
	Short: "Orchestrate a swarm of AI agents",
}

var swarmStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a swarm of worker agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		workers, _ := cmd.Flags().GetInt("workers")
		projectName, _ := cmd.Flags().GetString("project")

		if projectName == "" {
			var err error
			projectName, err = getCurrentProject()
			if err != nil {
				return fmt.Errorf("could not determine project: %w", err)
			}
		}

		// 1. Extract Scripts
		scriptDir, err := ExtractScripts()
		if err != nil {
			return fmt.Errorf("failed to extract scripts: %w", err)
		}
		
		if !globalJSON {
			fmt.Printf("Scripts extracted to %s\n", scriptDir)
		}

		// 2. Setup Bridge Directory
		bridgeDir := "/tmp"
		if _, err := os.Stat(bridgeDir); os.IsNotExist(err) {
			return fmt.Errorf("system bridge directory %s does not exist", bridgeDir)
		}

		// 3. Load Project and tasks to determine environments
		dataDir, _ := getDataDir()
		projectManager := NewProjectDataManager(dataDir, NewVersionManager(version))
		views, _, err := projectManager.GetTaskViews(projectName)
		if err != nil {
			return err
		}

		// 4. Start Workers
		runner := &BackgroundRunner{}
		
		if !globalJSON {
			fmt.Printf("Initializing Swarm for project '%s' with %d workers...\n", projectName, workers)
		}

		for i := 1; i <= workers; i++ {
			agentID := fmt.Sprintf("worker-%d", i)
			
			// Simple allocation: find next pending task or default to local
			var targetTask *TaskView
			for _, v := range views {
				if v.Status == "PENDING" && (v.AssignedTo == "" || v.AssignedTo == agentID) {
					targetTask = &v
					break
				}
			}
			
			if targetTask == nil {
				// Default task view for worker if none found
				targetTask = &TaskView{
					ID: "default",
					Behavior: AgentBehavior{
						Environment: EnvironmentConfig{Provider: "local"},
					},
				}
			}

			if err := runner.Start(projectName, agentID, scriptDir, targetTask); err != nil {
				fmt.Printf("Failed to start worker %d: %v\n", i, err)
			}
			time.Sleep(200 * time.Millisecond)
		}
		
		if globalJSON {
			fmt.Println("{\"status\": \"started\", \"workers\":", workers, "}")
		} else {
			fmt.Println("Swarm fully operational.")
		}

		// 5. Supervisor Loop (if enabled)
		supervisorEnabled, _ := cmd.Flags().GetBool("supervisor")
		if supervisorEnabled {
			if !globalJSON {
				fmt.Println("🛡️ Supervisor active. Monitoring for blocked agents...")
			}
			runSupervisor(projectName)
		}

		return nil
	},
}


func runSupervisor(projectName string) {
	dataDir, _ := getDataDir()
	versionManager := NewVersionManager(version)
	projectManager := NewProjectDataManager(dataDir, versionManager)

	// Determine which file to watch
	taskFile := filepath.Join(dataDir, projectName, "project.yaml")
	if _, err := os.Stat(taskFile); os.IsNotExist(err) {
		taskFile = filepath.Join(dataDir, projectName, "tasks.yaml")
	}

	fmt.Printf("🛡️ Supervisor: Watching %s for state transitions...\n", taskFile)

	for {
		views, _, err := projectManager.GetTaskViews(projectName)
		if err == nil {
			for _, task := range views {
				if task.Status == "BLOCKED" {
					fmt.Printf("🛡️ Supervisor: Handling BLOCKED Task %s\n", task.ID)
					
					// 1. Generate Remedy
					healTaskText := fmt.Sprintf("REMEDY: Resolve blocker in Task %s", task.ID)
					
					// 2. Inject (v1.1 or legacy handled by manager)
					if task.IsV11 {
						v11, _ := projectManager.LoadProjectV11(projectName)
						v11.Tasks = append(v11.Tasks, TaskV11{
							ID:     fmt.Sprintf("remedy-%d", time.Now().Unix()),
							Name:   healTaskText,
							Status: "PENDING",
							Behavior: AgentBehavior{
								Role: "Senior Troubleshooter",
							},
							UpdatedAt: time.Now(),
						})
						projectManager.SaveProjectV11(projectName, v11)
					} else {
						legacy, _ := projectManager.LoadProjectData(projectName)
						maxID := 0
						for _, t := range legacy.Tasks {
							if t.ID > maxID { maxID = t.ID }
						}
						legacy.Tasks = append(legacy.Tasks, Task{
							ID:      maxID + 1,
							Text:    healTaskText,
							Created: time.Now(),
							Behavior: AgentBehavior{Role: "Senior Troubleshooter"},
						})
						projectManager.SaveProjectData(projectName, legacy)
					}
					fmt.Printf("🛡️ Supervisor: Injected remedy for %s\n", task.ID)
				}
			}
		}

		// Wait for file change instead of fixed 10s sleep
		exec.Command("inotifywait", "-q", "-e", "modify", taskFile).Run()
		// Small cooldown to prevent rapid fire
		time.Sleep(500 * time.Millisecond)
	}
}

func init() {
	swarmCmd.AddCommand(swarmStartCmd)
	swarmStartCmd.Flags().IntP("workers", "w", 3, "Number of worker agents to spawn")
	swarmStartCmd.Flags().StringP("project", "p", "", "Project name")
	swarmStartCmd.Flags().Bool("supervisor", false, "Enable the Self-Healing Supervisor")
}
