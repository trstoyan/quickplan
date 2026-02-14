package main

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var (
	deleteCmd = &cobra.Command{
		Use:   "delete [task-id]",
		Short: "Delete a task from the current or specified project",
		Long: `Delete a task by its ID. Requires confirmation unless --force flag is used.
If --project flag is provided, deletes from that project instead of the current context project.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task ID
			taskID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}

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

			// Find task by ID
			taskIndex := -1
			var taskToDelete *Task
			for i, task := range projectData.Tasks {
				if task.ID == taskID {
					taskIndex = i
					taskToDelete = &task
					break
				}
			}

			if taskIndex == -1 {
				return fmt.Errorf("task with ID %d not found in project '%s'", taskID, targetProject)
			}

			// Check for force flag
			forceFlag, _ := cmd.Flags().GetBool("force")

			// Request confirmation unless --force is used
			if !forceFlag {
				confirmed, err := confirmDeletion(taskToDelete)
				if err != nil {
					return fmt.Errorf("failed to get confirmation: %w", err)
				}

				if !confirmed {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			// Delete task
			projectData.Tasks = append(projectData.Tasks[:taskIndex], projectData.Tasks[taskIndex+1:]...)

			// Renumber tasks to maintain sequential IDs
			renumberTasks(projectData.Tasks)

			// Save updated project data
			if err := projectManager.SaveProjectData(targetProject, projectData); err != nil {
				return fmt.Errorf("failed to save project data: %w", err)
			}

			fmt.Printf("Deleted task %d from project '%s': %s\n", taskID, targetProject, taskToDelete.Text)
			return nil
		},
	}
)

func init() {
	deleteCmd.Flags().StringP("project", "p", "", "Delete task from this project instead of current")
	deleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

// confirmDeletion displays a confirmation dialog for task deletion
func confirmDeletion(task *Task) (bool, error) {
	var confirmed bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Delete task: %s", task.Text)).
				Description("This action cannot be undone.").
				Value(&confirmed).
				Affirmative("Delete").
				Negative("Cancel"),
		),
	)

	if err := form.Run(); err != nil {
		return false, err
	}

	return confirmed, nil
}

// renumberTasks renumbers task IDs to be sequential starting from 1
// Following Single Responsibility Principle - handles only ID renumbering
func renumberTasks(tasks []Task) {
	for i := range tasks {
		tasks[i].ID = i + 1
	}
}

// getTargetProject determines the target project from flags or current context
// Following DRY principle - reusable across commands
func getTargetProject(cmd *cobra.Command) (string, error) {
	projectFlag, _ := cmd.Flags().GetString("project")
	if projectFlag != "" {
		return projectFlag, nil
	}

	targetProject, err := getCurrentProject()
	if err != nil {
		return "", fmt.Errorf("failed to get current project: %w", err)
	}

	return targetProject, nil
}
