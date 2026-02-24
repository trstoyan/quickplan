package main

import (
	"fmt"
	"strings"

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

var agentRunCmd = &cobra.Command{
	Use:   "run [task_id]",
	Short: "Execute the agent assigned to a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskIDStr := args[0]
		projectName, _ := cmd.Flags().GetString("project")
		if projectName == "" {
			projectName, _ = getCurrentProject()
		}

		dataDir, _ := getDataDir()
		projectManager := NewProjectDataManager(dataDir, NewVersionManager(version))

		views, _, err := projectManager.GetTaskViews(projectName)
		if err != nil {
			return err
		}

		var target *TaskView
		for _, v := range views {
			if v.ID == taskIDStr {
				target = &v
				break
			}
		}

		if target == nil {
			return fmt.Errorf("task %s not found", taskIDStr)
		}

		if strings.HasPrefix(target.AssignedTo, "plugin:") {
			pluginName := strings.TrimPrefix(target.AssignedTo, "plugin:")
			fmt.Printf("🔌 Executing plugin: %s\n", pluginName)

			req := PluginRequest{
				TaskID:   target.ID,
				Role:     target.Behavior.Role,
				Strategy: target.Behavior.Strategy,
			}

			resp, err := ExecutePlugin(pluginName, req)
			if err != nil {
				return err
			}

			fmt.Printf("Plugin Result (%s): %s\n", resp.Status, resp.Message)
			return nil
		}

		// Fallback: just output prompt
		fmt.Println(GenerateSystemPrompt(nil, projectName)) // Need actual Task for proper prompt
		return fmt.Errorf("no plugin assigned and automated LLM execution not configured for this task")
	},
}

func init() {
	agentCmd.AddCommand(agentInitCmd)
	agentCmd.AddCommand(agentRunCmd)
	agentInitCmd.Flags().StringP("project", "p", "", "Project name")
	agentRunCmd.Flags().StringP("project", "p", "", "Project name")
}
