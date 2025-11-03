package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
		
		tasksFile := filepath.Join(dataDir, targetProject, "tasks.yaml")
		
		data, err := os.ReadFile(tasksFile)
		if err != nil {
			return fmt.Errorf("failed to read tasks file: %w", err)
		}
		
		var projectData ProjectData
		if err := yaml.Unmarshal(data, &projectData); err != nil {
			return fmt.Errorf("failed to parse tasks file: %w", err)
		}
		
		// Toggle archive state
		projectData.Archived = !projectData.Archived
		projectData.Modified = time.Now()
		
		// Save to file
		data, err = yaml.Marshal(&projectData)
		if err != nil {
			return fmt.Errorf("failed to marshal tasks: %w", err)
		}
		
		if err := os.WriteFile(tasksFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write tasks file: %w", err)
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
