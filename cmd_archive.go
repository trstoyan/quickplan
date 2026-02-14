package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive [project]",
	Short: "Archive or unarchive a project",
	Long: `Archive or unarchive a project by name. If no project name is provided,
archives the current project. Use --toggle to switch archive state.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetProject string

		if len(args) > 0 {
			targetProject = args[0]
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

		// Toggle archive state
		projectData.Archived = !projectData.Archived

		// Save project data
		if err := projectManager.SaveProjectData(targetProject, projectData); err != nil {
			return fmt.Errorf("failed to save project data: %w", err)
		}

		status := "archived"
		if !projectData.Archived {
			status = "unarchived"
		}
		fmt.Printf("Project '%s' has been %s\n", targetProject, status)
		return nil
	},
}

// Command registered in main.go
