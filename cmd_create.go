package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project",
	Long: `Create a new named project. If --project flag is provided, it creates
a project with that name. Otherwise, it creates a project with the given name
or defaults to creating/using 'default' project.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectName string

		// Get project name from flag or argument
		projectFlag, _ := cmd.Flags().GetString("project")
		if projectFlag != "" {
			projectName = projectFlag
		} else if len(args) > 0 {
			projectName = args[0]
		} else {
			projectName = "default"
		}

		dataDir, err := getDataDir()
		if err != nil {
			return fmt.Errorf("failed to get data directory: %w", err)
		}

		projectDir := filepath.Join(dataDir, projectName)

		// Check if project already exists
		if _, err := os.Stat(projectDir); err == nil {
			return fmt.Errorf("project '%s' already exists", projectName)
		}

		// Create project using ProjectDataManager
		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		if err := projectManager.CreateProject(projectName); err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		// Set as current project
		if err := setCurrentProject(projectName); err != nil {
			return fmt.Errorf("failed to set current project: %w", err)
		}

		fmt.Printf("Created project '%s' and set as current\n", projectName)
		return nil
	},
}

func init() {
	createCmd.Flags().StringP("project", "p", "", "Project name")
}
