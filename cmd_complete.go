package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete [task-id]",
	Short: "Mark a task as completed",
	Long: `Mark a task as completed by its ID. If no ID is provided,
displays an interactive menu to select a task to complete.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		
		// Filter out already completed tasks for menu
		var incompleteTasks []Task
		for _, task := range projectData.Tasks {
			if !task.Done {
				incompleteTasks = append(incompleteTasks, task)
			}
		}
		
		if len(incompleteTasks) == 0 {
			fmt.Println("No incomplete tasks found")
			return nil
		}
		
		var taskToComplete *Task
		
		if len(args) > 0 {
			// Task ID provided directly
			taskID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			
			// Find task by ID
			found := false
			for i := range projectData.Tasks {
				if projectData.Tasks[i].ID == taskID {
					if projectData.Tasks[i].Done {
						return fmt.Errorf("task %d is already completed", taskID)
					}
					taskToComplete = &projectData.Tasks[i]
					found = true
					break
				}
			}
			
			if !found {
				return fmt.Errorf("task %d not found", taskID)
			}
		} else {
			// Show interactive menu
			type taskChoice struct {
				label string
				taskID int
			}
			
			var choices []taskChoice
			for _, task := range incompleteTasks {
				choices = append(choices, taskChoice{
					label: fmt.Sprintf("%d. %s", task.ID, task.Text),
					taskID: task.ID,
				})
			}
			
			var selected taskChoice
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[taskChoice]().
						Title("Select task to complete").
						Options(huh.NewOptions(choices...)...).
						Value(&selected).
						Description("Navigate with arrow keys, press Enter to select"),
				),
			)
			
			if err := form.Run(); err != nil {
				return fmt.Errorf("failed to show menu: %w", err)
			}
			
			// Find task by ID in projectData.Tasks (not incompleteTasks)
			found := false
			for i := range projectData.Tasks {
				if projectData.Tasks[i].ID == selected.taskID {
					if projectData.Tasks[i].Done {
						return fmt.Errorf("task %d is already completed", selected.taskID)
					}
					taskToComplete = &projectData.Tasks[i]
					found = true
					break
				}
			}
			
			if !found {
				return fmt.Errorf("task %d not found", selected.taskID)
			}
		}
		
		// Mark task as completed
		taskToComplete.Done = true
		now := time.Now()
		taskToComplete.Completed = &now

		noteText, _ := cmd.Flags().GetString("note")
		if noteText != "" {
			taskToComplete.Notes = append(taskToComplete.Notes, NoteEntry{
				Text:      noteText,
				Timestamp: now,
			})
		}

		// Save project data
		if err := projectManager.SaveProjectData(targetProject, projectData); err != nil {
			return fmt.Errorf("failed to save project data: %w", err)
		}

		fmt.Printf("Completed task: %s\n", taskToComplete.Text)
		return nil
	},
}

func init() {
	completeCmd.Flags().StringP("project", "p", "", "Complete task in this project instead of current")
	completeCmd.Flags().StringP("note", "n", "", "Add a note to the completed task")
}
