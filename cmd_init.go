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
	Use:   "init [project_name]",
	Short: "Initialize a new Quick Plan project",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := ""
		if len(args) > 0 {
			projectName = args[0]
		}

		isInteractive, _ := cmd.Flags().GetBool("interactive")
		
		// If non-interactive is set globally, or if we have a name and not asking for interactive
		if globalNonInteractive || (!isInteractive && projectName != "") {
			if projectName == "" {
				projectName = "new-project"
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
			
			if globalJSON {
				fmt.Printf("{\"status\": \"success\", \"project\": \"%s\"}\n", projectName)
			} else {
				fmt.Printf("✓ Project '%s' initialized.\n", projectName)
			}
			return nil
		}

		if !isInteractive {
			return fmt.Errorf("use --interactive flag to run the wizard or provide a project name")
		}

		scanner := bufio.NewScanner(os.Stdin)


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
