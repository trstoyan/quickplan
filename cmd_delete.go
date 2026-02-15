package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	deleteCmd = &cobra.Command{
		Use:   "delete [task-id]...",
		Short: "Delete tasks from the current or specified project",
		Long: `Delete one or more tasks by their IDs. Requires confirmation unless --force flag is used.
If --project flag is provided, deletes from that project instead of the current context project.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse task IDs
			var taskIDs []int
			for _, arg := range args {
				id, err := strconv.Atoi(arg)
				if err != nil {
					return fmt.Errorf("invalid task ID: %s", arg)
				}
				taskIDs = append(taskIDs, id)
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

			// Create a deep copy for backup BEFORE deletion
			// Using YAML marshal/unmarshal as a quick way to deep copy
			projectDataBeforeDeletion := &ProjectData{}
			if yamlBytes, err := yaml.Marshal(projectData); err == nil {
				yaml.Unmarshal(yamlBytes, projectDataBeforeDeletion)
			}

			// Find tasks by ID
			var tasksToDelete []Task
			var indicesToDelete []int
			var missingIDs []int

			for _, id := range taskIDs {
				found := false
				for i, task := range projectData.Tasks {
					if task.ID == id {
						indicesToDelete = append(indicesToDelete, i)
						tasksToDelete = append(tasksToDelete, task)
						found = true
						break
					}
				}
				if !found {
					missingIDs = append(missingIDs, id)
				}
			}

			if len(missingIDs) > 0 {
				return fmt.Errorf("tasks with IDs %v not found in project '%s'", missingIDs, targetProject)
			}

			// Check for force flag
			forceFlag, _ := cmd.Flags().GetBool("force")

			// Request confirmation unless --force is used
			if !forceFlag {
				confirmed, err := confirmDeletions(tasksToDelete)
				if err != nil {
					return fmt.Errorf("failed to get confirmation: %w", err)
				}

				if !confirmed {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			// Delete tasks (in reverse order to maintain correct indices)
			// First, sort indices to delete in descending order
			for i := 0; i < len(indicesToDelete); i++ {
				for j := i + 1; j < len(indicesToDelete); j++ {
					if indicesToDelete[i] < indicesToDelete[j] {
						indicesToDelete[i], indicesToDelete[j] = indicesToDelete[j], indicesToDelete[i]
					}
				}
			}

			for _, idx := range indicesToDelete {
				projectData.Tasks = append(projectData.Tasks[:idx], projectData.Tasks[idx+1:]...)
			}

			// Save backup for undo
			undoData := struct {
				ProjectName string      `yaml:"project_name"`
				Data        ProjectData `yaml:"data"`
			}{
				ProjectName: targetProject,
				Data:        *projectDataBeforeDeletion,
			}
			undoBackupPath := filepath.Join(dataDir, ".undo_backup.yaml")
			if undoBytes, err := yaml.Marshal(undoData); err == nil {
				os.WriteFile(undoBackupPath, undoBytes, 0644)
			}

			// Renumber tasks to maintain sequential IDs
			renumberTasks(projectData.Tasks)

			// Save updated project data
			if err := projectManager.SaveProjectData(targetProject, projectData); err != nil {
				return fmt.Errorf("failed to save project data: %w", err)
			}

			if len(tasksToDelete) == 1 {
				fmt.Printf("Deleted task %d from project '%s': %s\n", taskIDs[0], targetProject, tasksToDelete[0].Text)
			} else {
				fmt.Printf("Deleted %d tasks from project '%s'\n", len(tasksToDelete), targetProject)
			}
			fmt.Println("Tip: Use 'quickplan undo' to restore deleted tasks.")
			return nil
		},
	}
)

func init() {
	deleteCmd.Flags().StringP("project", "p", "", "Delete task from this project instead of current")
	deleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

// confirmDeletions displays a confirmation dialog for multiple task deletions
func confirmDeletions(tasks []Task) (bool, error) {
	var confirmed bool
	var title string
	var description string

	if len(tasks) == 1 {
		title = fmt.Sprintf("Delete task: %s", tasks[0].Text)
		description = "This action cannot be undone."
	} else {
		title = fmt.Sprintf("Delete %d tasks?", len(tasks))
		description = "Tasks to delete:\n"
		for _, t := range tasks {
			description += fmt.Sprintf("- %s\n", t.Text)
		}
		description += "\nThis action cannot be undone."
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
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
