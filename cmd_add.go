package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	addCmd = &cobra.Command{
		Use:   "add [task]",
		Short: "Add a task to the current or specified project",
		Long: `Add a new task to your project. If --project flag is provided,
adds the task to that project instead of the current context project.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskText := args[0]
			
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
			
			// Load existing tasks
			dataDir, err := getDataDir()
			if err != nil {
				return fmt.Errorf("failed to get data directory: %w", err)
			}
			
			tasksFile := filepath.Join(dataDir, targetProject, "tasks.yaml")
			
			var projectData ProjectData
			
			// Read existing file if it exists
			if data, err := os.ReadFile(tasksFile); err == nil {
				if err := yaml.Unmarshal(data, &projectData); err != nil {
					return fmt.Errorf("failed to parse tasks file: %w", err)
				}
			}
			
			// Add new task
			newTask := Task{
				ID:       len(projectData.Tasks) + 1,
				Text:     taskText,
				Done:     false,
				Created:  time.Now(),
				Completed: nil,
			}
			projectData.Tasks = append(projectData.Tasks, newTask)
			
			// Update modified timestamp
			projectData.Modified = time.Now()
			
			// Save to file
			data, err := yaml.Marshal(&projectData)
			if err != nil {
				return fmt.Errorf("failed to marshal tasks: %w", err)
			}
			
			if err := os.WriteFile(tasksFile, data, 0644); err != nil {
				return fmt.Errorf("failed to write tasks file: %w", err)
			}
			
			fmt.Printf("Added task to project '%s': %s\n", targetProject, taskText)
			return nil
		},
	}
)

func init() {
	addCmd.Flags().StringP("project", "p", "", "Add task to this project instead of current")
}
