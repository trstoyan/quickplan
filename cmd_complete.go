package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var completeCmd = &cobra.Command{
	Use:   "complete [task-id]",
	Short: "Mark a task as completed",
	Long: `Mark a task as completed by its ID. If no ID is provided,
displays an interactive menu to select a task to complete.`,
	Args: cobra.MaximumNArgs(1),
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
				task  *Task
			}
			
			var choices []taskChoice
			for i := range incompleteTasks {
				choices = append(choices, taskChoice{
					label: fmt.Sprintf("%d. %s", incompleteTasks[i].ID, incompleteTasks[i].Text),
					task:  &incompleteTasks[i],
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
			
			taskToComplete = selected.task
		}
		
		// Mark task as completed
		taskToComplete.Done = true
		now := time.Now()
		taskToComplete.Completed = &now
		
		// Update modified timestamp
		projectData.Modified = time.Now()
		
		// Save to file
		data, err = yaml.Marshal(&projectData)
		if err != nil {
			return fmt.Errorf("failed to marshal tasks: %w", err)
		}
		
		if err := os.WriteFile(tasksFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write tasks file: %w", err)
		}
		
		fmt.Printf("Completed task: %s\n", taskToComplete.Text)
		return nil
	},
}

func init() {
	completeCmd.Flags().StringP("project", "p", "", "Complete task in this project instead of current")
}
