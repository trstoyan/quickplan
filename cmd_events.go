package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "View project events",
}

var eventsTailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Show the latest events for the current project",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		eventLog, err := projectManager.LoadEvents(projectName)
		if err != nil {
			return err
		}

		n, _ := cmd.Flags().GetInt("n")
		events := eventLog.Events
		if len(events) > n {
			events = events[len(events)-n:]
		}

		if len(events) == 0 {
			fmt.Println("No events found.")
			return nil
		}

		for _, e := range events {
			fmt.Printf("[%s] %s: %s", e.Timestamp.Format("2006-01-02 15:04:05"), e.Type, e.Actor)
			if e.TaskID != "" {
				fmt.Printf(" (Task: %s)", e.TaskID)
			}
			if e.PrevStatus != "" || e.NextStatus != "" {
				fmt.Printf(" %s -> %s", e.PrevStatus, e.NextStatus)
			}
			if e.Message != "" {
				fmt.Printf(" - %s", e.Message)
			}
			fmt.Println()
		}

		return nil
	},
}

var eventsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export events as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		eventLog, err := projectManager.LoadEvents(projectName)
		if err != nil {
			return err
		}

		out, err := json.MarshalIndent(eventLog.Events, "", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(out))
		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsTailCmd)
	eventsTailCmd.Flags().IntP("n", "n", 50, "Number of events to show")
	eventsTailCmd.Flags().StringP("project", "p", "", "Project name")

	eventsCmd.AddCommand(eventsExportCmd)
	eventsExportCmd.Flags().Bool("json", true, "Export as JSON (default true)")
	eventsExportCmd.Flags().StringP("project", "p", "", "Project name")
}
