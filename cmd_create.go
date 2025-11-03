package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
		
		// Create project directory
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}
		
		// Create tasks.yaml file with timestamps
		tasksFile := filepath.Join(projectDir, "tasks.yaml")
		now := time.Now()
		projectData := ProjectData{
			Tasks:    []Task{},
			Created:  now,
			Modified: now,
			Archived: false,
		}
		
		data, err := yaml.Marshal(&projectData)
		if err != nil {
			return fmt.Errorf("failed to marshal project data: %w", err)
		}
		
		if err := os.WriteFile(tasksFile, data, 0644); err != nil {
			return fmt.Errorf("failed to create tasks file: %w", err)
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
