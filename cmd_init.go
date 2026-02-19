package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive wizard to create a new Quick Plan project",
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive, _ := cmd.Flags().GetBool("interactive")
		if !interactive {
			return fmt.Errorf("use --interactive flag to run the wizard")
		}

		scanner := bufio.NewScanner(os.Stdin)

		fmt.Println("🤖 Welcome to the Quick Plan Init Wizard")
		fmt.Println("---------------------------------------")

		// 1. Project Name
		fmt.Print("Enter project name: ")
		if !scanner.Scan() {
			return fmt.Errorf("input error")
		}
		projectName := strings.TrimSpace(scanner.Text())
		if projectName == "" {
			return fmt.Errorf("project name cannot be empty")
		}

		dataDir, err := getDataDir()
		if err != nil {
			return err
		}
		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		if err := projectManager.CreateProject(projectName); err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}
		if err := setCurrentProject(projectName); err != nil {
			return fmt.Errorf("failed to set current project: %w", err)
		}
		fmt.Printf("✓ Project '%s' created.\n", projectName)

		// 2. Initial Tasks
		fmt.Println("\nLet's add some initial tasks.")
		fmt.Println("Enter tasks one by one (empty line to finish):")
		
		projectData, err := projectManager.LoadProjectData(projectName)
		if err != nil {
			return err
		}

		count := 1
		for {
			fmt.Printf("Task %d: ", count)
			if !scanner.Scan() {
				break
			}
			taskText := strings.TrimSpace(scanner.Text())
			if taskText == "" {
				break
			}
			
			fmt.Print("  Assign Role (default: Generalist): ")
			if !scanner.Scan() {
				break
			}
			role := strings.TrimSpace(scanner.Text())
			if role == "" {
				role = "Generalist"
			}
			
			newTask := Task{
				ID:        len(projectData.Tasks) + 1,
				Text:      taskText,
				Behavior: AgentBehavior{
					Role: role,
				},
				Created: time.Now(),
			}
			projectData.Tasks = append(projectData.Tasks, newTask)
			count++
		}

		if err := projectManager.SaveProjectData(projectName, projectData); err != nil {
			return err
		}

		fmt.Printf("\n✓ Added %d tasks to '%s'.\n", count-1, projectName)
		fmt.Println("You can now start the swarm with: quickplan swarm start --project " + projectName)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolP("interactive", "i", false, "Run in interactive mode")
}
