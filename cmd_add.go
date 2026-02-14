package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
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
			targetProject, err := getTargetProject(cmd)
			if err != nil {
				return err
			}

			// Validate project exists
			if !projectExists(targetProject) {
				return fmt.Errorf("project '%s' does not exist", targetProject)
			}

			// Load project data
			dataDir, err := getDataDir()
			if err != nil {
				return fmt.Errorf("failed to get data directory: %w", err)
			}

			versionManager := NewVersionManager(version)
			projectManager := NewProjectDataManager(dataDir, versionManager)

			projectData, err := projectManager.LoadProjectData(targetProject)
			if err != nil {
				return fmt.Errorf("failed to load project data: %w", err)
			}

			// Add new task
			newTask := Task{
				ID:        len(projectData.Tasks) + 1,
				Text:      taskText,
				Done:      false,
				Created:   time.Now(),
				Completed: nil,
			}
			projectData.Tasks = append(projectData.Tasks, newTask)

			// Save project data
			if err := projectManager.SaveProjectData(targetProject, projectData); err != nil {
				return fmt.Errorf("failed to save project data: %w", err)
			}

			fmt.Printf("Added task to project '%s': %s\n", targetProject, taskText)
			return nil
		},
	}
)

func init() {
	addCmd.Flags().StringP("project", "p", "", "Add task to this project instead of current")
}
