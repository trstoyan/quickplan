package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	for {
		projectData, err := projectManager.LoadProjectData(projectName)
		if err == nil {
			for _, task := range projectData.Tasks {
				// Simple heuristic: if a task has a blocker note or is marked in a specific way
				// In this protocol, we'll look for tasks that are not done and have "BLOCKED" in notes
				isBlocked := false
				var blockerReason string
				for _, note := range task.Notes {
					if strings.Contains(strings.ToUpper(note.Text), "BLOCKED") {
						isBlocked = true
						blockerReason = note.Text
						break
					}
				}

				if isBlocked && !task.Done {
					fmt.Printf("🛡️ Supervisor: Detected blocker in Task %d: %s\n", task.ID, blockerReason)
					
					// Initialize Correction Agent logic
					// 1. Generate a "Heal" task
					healTask := Task{
						ID:        len(projectData.Tasks) + 1,
						Text:      fmt.Sprintf("REMEDY: Resolve blocker in Task %d: %s", task.ID, blockerReason),
						Created:   time.Now(),
						Behavior: AgentBehavior{
							Role:     "Senior Troubleshooter",
							Strategy: "Recursive Debugging",
						},
					}
					
					// 2. Inject into YAML
					projectData.Tasks = append(projectData.Tasks, healTask)
					projectManager.SaveProjectData(projectName, projectData)
					
					fmt.Printf("🛡️ Supervisor: Injected Remedy Task %d into the plan.\n", healTask.ID)
					
					// To avoid infinite loops, we should mark the original task or note as "HANDLED"
					// For now, we'll just sleep to poll
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func init() {
	swarmCmd.AddCommand(swarmStartCmd)
	swarmStartCmd.Flags().IntP("workers", "w", 3, "Number of worker agents to spawn")
	swarmStartCmd.Flags().StringP("project", "p", "", "Project name")
	swarmStartCmd.Flags().Bool("supervisor", false, "Enable the Self-Healing Supervisor")
}
