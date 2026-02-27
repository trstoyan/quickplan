package main

import (
	"encoding/json"
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

		// Try v1.1 first
		if _, err := projectManager.LoadProjectV11(targetProject); err == nil {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required for v1.1 projects")
			}
			taskID := args[0]

			views, _, err := projectManager.GetTaskViews(targetProject)
			if err != nil {
				return err
			}

			prevStatus := ""
			found := false
			for _, v := range views {
				if v.ID == taskID {
					prevStatus = v.Status
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("task %s not found", taskID)
			}

			if err := completeTaskWithTransitions(projectManager, targetProject, taskID, prevStatus, "human"); err != nil {
				return err
			}

			SendPulse(targetProject, "human", taskID, "DONE", prevStatus)

			if globalJSON {
				output := map[string]interface{}{
					"status":  "success",
					"project": targetProject,
					"task": map[string]interface{}{
						"id":     taskID,
						"status": "DONE",
						"done":   true,
					},
				}
				payload, _ := json.Marshal(output)
				fmt.Println(string(payload))
				return nil
			}

			fmt.Printf("Completed task: %s (v1.1)\n", taskID)
			return nil
		}

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

		var selectedTaskID int
		var prevStatus string

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
					selectedTaskID = taskID
					prevStatus = GetTaskStatus(projectData.Tasks[i])
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
				label  string
				taskID int
			}

			var choices []taskChoice
			for _, task := range incompleteTasks {
				choices = append(choices, taskChoice{
					label:  fmt.Sprintf("%d. %s", task.ID, task.Text),
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
					selectedTaskID = selected.taskID
					prevStatus = GetTaskStatus(projectData.Tasks[i])
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("task %d not found", selected.taskID)
			}
		}

		taskIDKey := fmt.Sprintf("t-%d", selectedTaskID)
		if err := completeTaskWithTransitions(projectManager, targetProject, taskIDKey, prevStatus, "human"); err != nil {
			return err
		}

		noteText, _ := cmd.Flags().GetString("note")
		if noteText != "" {
			reloaded, err := projectManager.LoadProjectData(targetProject)
			if err != nil {
				return fmt.Errorf("failed to reload project data for note update: %w", err)
			}
			for i := range reloaded.Tasks {
				if reloaded.Tasks[i].ID == selectedTaskID {
					reloaded.Tasks[i].Notes = append(reloaded.Tasks[i].Notes, NoteEntry{
						Text:      noteText,
						Timestamp: time.Now(),
					})
					break
				}
			}
			if err := projectManager.SaveProjectData(targetProject, reloaded); err != nil {
				return fmt.Errorf("failed to save note update: %w", err)
			}
		}

		// Emit pulse
		SendPulse(targetProject, "human", selectedTaskID, "DONE", prevStatus)

		if globalJSON {
			output := map[string]interface{}{
				"status":  "success",
				"project": targetProject,
				"task": map[string]interface{}{
					"id":     selectedTaskID,
					"status": "DONE",
					"done":   true,
				},
			}
			payload, _ := json.Marshal(output)
			fmt.Println(string(payload))
			return nil
		}

		fmt.Printf("Completed task: %d\n", selectedTaskID)
		return nil
	},
}

func init() {
	completeCmd.Flags().StringP("project", "p", "", "Complete task in this project instead of current")
	completeCmd.Flags().StringP("note", "n", "", "Add a note to the completed task")
}

func completeTaskWithTransitions(projectManager *ProjectDataManager, projectName, taskID, currentStatus, actor string) error {
	current := canonicalStatus(currentStatus)
	if current == "DONE" {
		return fmt.Errorf("task %s is already completed", taskID)
	}

	switch current {
	case "PENDING":
		if err := projectManager.UpdateTaskStatus(projectName, taskID, "IN_PROGRESS", actor); err != nil {
			return err
		}
	case "IN_PROGRESS":
		// already in execution state
	case "BLOCKED":
		return fmt.Errorf("task %s is BLOCKED; resolve dependencies/guards first", taskID)
	default:
		return fmt.Errorf("task %s cannot be completed from status %s", taskID, currentStatus)
	}

	if err := projectManager.UpdateTaskStatus(projectName, taskID, "DONE", actor); err != nil {
		return err
	}

	return nil
}
