package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "AI agent related commands",
}

var agentInitCmd = &cobra.Command{
	Use:   "init [task_id]",
	Short: "Initialize an agent for a specific task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskIDStr := args[0]
		
		// Parse task ID
		var taskID int
		if _, err := fmt.Sscanf(taskIDStr, "%d", &taskID); err != nil {
			return fmt.Errorf("invalid task ID: %s", taskIDStr)
		}

		targetProject, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		projectData, err := projectManager.LoadProjectData(targetProject)
		if err != nil {
			return err
		}

		var targetTask *Task
		for _, t := range projectData.Tasks {
			if t.ID == taskID {
				targetTask = &t
				break
			}
		}

		if targetTask == nil {
			return fmt.Errorf("task %d not found in project %s", taskID, targetProject)
		}

		prompt := GenerateSystemPrompt(targetTask, targetProject)
		fmt.Println(prompt)
		return nil
	},
}

func init() {
	agentCmd.AddCommand(agentInitCmd)
	agentInitCmd.Flags().StringP("project", "p", "", "Project name")
}
