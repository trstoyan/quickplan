package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks in the current or specified project",
	Long: `List all tasks in the current project, or in a specified project
using the --project flag. Shows task ID, text, and completion status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine target project
		var targetProject string
		projectFlag, _ := cmd.Flags().GetString("project")
		if projectFlag != "" {
			targetProject = projectFlag
		} else {
			var err error
			targetProject, err = getCurrentProject()
			if err != nil {
				return fmt.Errorf("failed to get current project: %w", err)
			}
		}
		
		// Validate project exists
		if !projectExists(targetProject) {
			return fmt.Errorf("project '%s' does not exist", targetProject)
		}
		
		// Load tasks
		dataDir, err := getDataDir()
		if err != nil {
			return fmt.Errorf("failed to get data directory: %w", err)
		}
		
		tasksFile := filepath.Join(dataDir, targetProject, "tasks.yaml")
		
		data, err := os.ReadFile(tasksFile)
		if err != nil {
			return fmt.Errorf("failed to read tasks file: %w", err)
		}
		
		var projectData ProjectData
		if err := yaml.Unmarshal(data, &projectData); err != nil {
			return fmt.Errorf("failed to parse tasks file: %w", err)
		}
		
		// Display tasks
		showAll, _ := cmd.Flags().GetBool("all")
		
		fmt.Printf("Tasks in project '%s':\n", targetProject)
		if projectData.Archived {
			fmt.Println("  [ARCHIVED]")
		}
		fmt.Println()
		
		if len(projectData.Tasks) == 0 {
			fmt.Println("  No tasks yet. Add one with 'quickplan add <task>'")
			return nil
		}
		
		var displayTasks []Task
		for _, task := range projectData.Tasks {
			if showAll || !task.Done {
				displayTasks = append(displayTasks, task)
			}
		}
		
		if !showAll && len(displayTasks) == 0 {
			fmt.Println("  All tasks completed! Use --all to see completed tasks.")
			return nil
		}
		
		for _, task := range displayTasks {
			status := "[ ]"
			if task.Done {
				status = "[✓]"
			}
			fmt.Printf("  %d. %s %s", task.ID, status, task.Text)
			
			// Show completion date if done
			if task.Done && task.Completed != nil {
				fmt.Printf(" (completed: %s)", task.Completed.Format("2006-01-02"))
			}
			fmt.Println()
		}
		
		return nil
	},
}

func init() {
	listCmd.Flags().StringP("project", "p", "", "List tasks from this project instead of current")
	listCmd.Flags().BoolP("all", "a", false, "Show all tasks including completed ones")
}
