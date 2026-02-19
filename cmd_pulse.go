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
		taskID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid task ID: %s", args[0])
		}
		status := args[1]
		
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			project, _ = getCurrentProject()
		}

		agentID := os.Getenv("AGENT_ID")
		if agentID == "" {
			agentID = "anonymous-agent"
		}

		SendPulse(project, agentID, taskID, status)
		fmt.Printf("📡 Pulse sent: Task %d is %s\n", taskID, status)
		return nil
	},
}

func init() {
	pulseCmd.Flags().StringP("project", "p", "", "Project name")
}
