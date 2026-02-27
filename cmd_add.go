package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	addCmd = &cobra.Command{
		Use:   "add [task]",
		Short: "Add a task to the current or specified project",
		Long: `Add a new task to your project. If --project flag is provided,
adds the task to that project instead of the current context project.

Note: In bash, ! triggers history expansion even inside double quotes.
Wrap the task in single quotes or escape ! if your text includes it.`,
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

			// Try v1.1 first
			if v11, err := projectManager.LoadProjectV11(targetProject); err == nil {
				// Parse flags
				assignedTo, _ := cmd.Flags().GetString("assigned-to")
				dependsOnRaw, _ := cmd.Flags().GetIntSlice("depends-on")
				role, _ := cmd.Flags().GetString("role")
				lifecycle, _ := cmd.Flags().GetString("lifecycle")
				strategy, _ := cmd.Flags().GetString("strategy")
				watchPath, _ := cmd.Flags().GetString("watch-path")

				// Map depends_on
				deps := make([]string, len(dependsOnRaw))
				for i, d := range dependsOnRaw {
					deps[i] = fmt.Sprintf("t-%d", d)
				}

				// Generate ID from max numeric suffix to avoid collisions after deletions.
				newTask := TaskV11{
					ID:         fmt.Sprintf("t-%d", nextV11TaskNumericID(v11.Tasks)),
					Name:       taskText,
					Status:     "TODO",
					AssignedTo: assignedTo,
					DependsOn:  deps,
					Behavior: AgentBehavior{
						Role:      role,
						LifeCycle: lifecycle,
						Strategy:  strategy,
					},
					Watch: WatchConfig{
						Paths: []string{watchPath},
					},
					UpdatedAt: time.Now(),
				}
				v11.Tasks = append(v11.Tasks, newTask)
				if err := projectManager.SaveProjectV11(targetProject, v11); err != nil {
					return fmt.Errorf("failed to save project v1.1: %w", err)
				}

				// Emit pulse
				SendPulse(targetProject, "human", newTask.ID, "TODO", "")

				if globalJSON {
					output := map[string]interface{}{
						"status":  "success",
						"project": targetProject,
						"task": map[string]interface{}{
							"id":     newTask.ID,
							"text":   newTask.Name,
							"status": newTask.Status,
							"done":   newTask.Status == "DONE",
						},
					}
					payload, _ := json.Marshal(output)
					fmt.Println(string(payload))
					return nil
				}

				fmt.Printf("Added task to project '%s' (v1.1): %s\n", targetProject, taskText)
				return nil
			}

			projectData, err := projectManager.LoadProjectData(targetProject)
			if err != nil {
				return fmt.Errorf("failed to load project data: %w", err)
			}

			// Parse flags
			assignedTo, _ := cmd.Flags().GetString("assigned-to")
			dependsOnRaw, _ := cmd.Flags().GetIntSlice("depends-on")
			role, _ := cmd.Flags().GetString("role")
			lifecycle, _ := cmd.Flags().GetString("lifecycle")
			strategy, _ := cmd.Flags().GetString("strategy")
			watchPath, _ := cmd.Flags().GetString("watch-path")

			// Add new task
			maxID := 0
			for _, t := range projectData.Tasks {
				if t.ID > maxID {
					maxID = t.ID
				}
			}

			newTask := Task{
				ID:         maxID + 1,
				Text:       taskText,
				Done:       false,
				Status:     "TODO",
				Created:    time.Now(),
				Completed:  nil,
				AssignedTo: assignedTo,
				DependsOn:  dependsOnRaw,
				Behavior: AgentBehavior{
					Role:      role,
					LifeCycle: lifecycle,
					Strategy:  strategy,
				},
				WatchPath: watchPath,
			}
			projectData.Tasks = append(projectData.Tasks, newTask)

			// Save project data
			if err := projectManager.SaveProjectData(targetProject, projectData); err != nil {
				return fmt.Errorf("failed to save project data: %w", err)
			}

			// Emit event
			projectManager.AppendEvent(targetProject, Event{
				Timestamp:  time.Now(),
				Type:       "TASK_CREATED",
				Actor:      "human",
				TaskID:     fmt.Sprintf("t-%d", newTask.ID),
				NextStatus: "TODO",
				Message:    fmt.Sprintf("Task created: %s", taskText),
			})

			// Emit pulse
			SendPulse(targetProject, "human", newTask.ID, "TODO", "")

			if globalJSON {
				output := map[string]interface{}{
					"status":  "success",
					"project": targetProject,
					"task": map[string]interface{}{
						"id":     newTask.ID,
						"text":   newTask.Text,
						"status": GetTaskStatus(newTask),
						"done":   newTask.Done,
					},
				}
				payload, _ := json.Marshal(output)
				fmt.Println(string(payload))
				return nil
			}

			fmt.Printf("Added task to project '%s': %s\n", targetProject, taskText)
			return nil
		},
	}
)

func init() {
	addCmd.Flags().StringP("project", "p", "", "Add task to this project instead of current")
	addCmd.Flags().String("assigned-to", "", "Assign task to agent or user")
	addCmd.Flags().IntSlice("depends-on", []int{}, "Comma-separated list of task IDs this task depends on")
	addCmd.Flags().String("role", "", "Role for the agent behavior")
	addCmd.Flags().String("lifecycle", "", "Lifecycle for the agent behavior (e.g., Atomic, Infinite)")
	addCmd.Flags().String("strategy", "", "Strategy for the agent behavior (e.g., TDD, Fast Prototype)")
	addCmd.Flags().String("watch-path", "", "Physical file path to watch for dependency verification")
}

func nextV11TaskNumericID(tasks []TaskV11) int {
	maxID := 0
	for _, t := range tasks {
		raw := strings.TrimPrefix(t.ID, "t-")
		if id, err := strconv.Atoi(raw); err == nil && id > maxID {
			maxID = id
		}
	}
	return maxID + 1
}
