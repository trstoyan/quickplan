package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show project statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		dataDir, _ := getDataDir()
		projectManager := NewProjectDataManager(dataDir, NewVersionManager(version))

		views, _, err := projectManager.GetTaskViews(projectName)
		if err != nil {
			return fmt.Errorf("failed to load tasks: %w", err)
		}

		counts := make(map[string]int)
		for _, v := range views {
			counts[v.Status]++
		}

		fmt.Printf("📊 Statistics for project '%s':\n", projectName)
		fmt.Printf("  Total Tasks:  %d\n", len(views))
		fmt.Printf("  Pending:      %d\n", counts["PENDING"])
		fmt.Printf("  In Progress:  %d\n", counts["IN_PROGRESS"])
		fmt.Printf("  Done:         %d\n", counts["DONE"])
		fmt.Printf("  Blocked:      %d\n", counts["BLOCKED"])
		fmt.Printf("  Failed:       %d\n", counts["FAILED"])

		// Event counts
		eventLog, err := projectManager.LoadEvents(projectName)
		if err == nil {
			fmt.Printf("  Total Events: %d\n", len(eventLog.Events))
		} else {
			// Check v1.1 embedded events
			v11, err := projectManager.LoadProjectV11(projectName)
			if err == nil {
				fmt.Printf("  Total Events: %d (embedded)\n", len(v11.Events))
			}
		}

		return nil
	},
}

func init() {
	statsCmd.Flags().StringP("project", "p", "", "Project name")
}
