package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var pulseCmd = &cobra.Command{
	Use:   "pulse [task_id] [status]",
	Short: "Send a status pulse to the swarm dashboard",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskIDStr := args[0]
		status := args[1]
		prevStatus, _ := cmd.Flags().GetString("prev-status")

		var taskID interface{}
		if id, err := strconv.Atoi(taskIDStr); err == nil {
			taskID = id
		} else {
			taskID = taskIDStr
		}

		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			project, _ = getCurrentProject()
		}

		agentID := os.Getenv("AGENT_ID")
		if agentID == "" {
			agentID = "anonymous-agent"
		}

		SendPulse(project, agentID, taskID, status, prevStatus)
		fmt.Printf("📡 Pulse sent: Task %v is %s (prev: %s)\n", taskID, status, prevStatus)
		return nil
	},
}

func init() {
	pulseCmd.Flags().StringP("project", "p", "", "Project name")
	pulseCmd.Flags().String("prev-status", "", "Previous status of the task")
}
