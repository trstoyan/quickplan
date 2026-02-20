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

func (br *BackgroundRunner) Start(project, agentID, scriptPath string) error {
	cmd := exec.Command(filepath.Join(scriptPath, "qp-loop.sh"), project, agentID)
	// We detach the process so it keeps running
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Printf("Started agent %s (PID: %d)\n", agentID, cmd.Process.Pid)
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
		fmt.Printf("Scripts extracted to %s\n", scriptDir)

		// 2. Setup Bridge Directory if needed (usually handled by script, but ensure base)
		bridgeDir := "/tmp"
		if _, err := os.Stat(bridgeDir); os.IsNotExist(err) {
			return fmt.Errorf("system bridge directory %s does not exist", bridgeDir)
		}

		// 3. Start Workers
		runner := &BackgroundRunner{}
		var wg sync.WaitGroup

		fmt.Printf("Initializing Swarm for project '%s' with %d workers...\n", projectName, workers)

		for i := 1; i <= workers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				agentID := fmt.Sprintf("worker-%d", id)
				if err := runner.Start(projectName, agentID, scriptDir); err != nil {
					fmt.Printf("Failed to start worker %d: %v\n", id, err)
				}
				// Stagger start slightly to avoid race conditions on pipe creation if any
				time.Sleep(200 * time.Millisecond)
			}(i)
		}
		
		wg.Wait()
		fmt.Println("Swarm fully operational.")

		// 4. Supervisor Loop (if enabled)
		supervisorEnabled, _ := cmd.Flags().GetBool("supervisor")
		if supervisorEnabled {
			fmt.Println("🛡️ Supervisor active. Monitoring for blocked agents...")
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
