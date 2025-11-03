package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var changeCmd = &cobra.Command{
	Use:   "change [project]",
	Short: "Change the current project context",
	Long: `Change the active project context. If no project name is provided,
displays a vim-inspired selection menu to choose from available projects.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var selectedProject string
		
		if len(args) > 0 {
			// Direct project name provided
			selectedProject = args[0]
		} else {
			// Show interactive menu
			projects, err := listProjects()
			if err != nil {
				return fmt.Errorf("failed to list projects: %w", err)
			}
			
			if len(projects) == 0 {
				return fmt.Errorf("no projects found. Create one with 'quickplan create <name>'")
			}
			
			var chosen string
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Select a project").
						Options(huh.NewOptions(projects...)...).
						Value(&chosen).
						Description("Navigate with arrow keys, press Enter to select"),
				),
			)
			
			if err := form.Run(); err != nil {
				return fmt.Errorf("failed to show menu: %w", err)
			}
			
			selectedProject = chosen
		}
		
		// Validate project exists
		if !projectExists(selectedProject) {
			return fmt.Errorf("project '%s' does not exist", selectedProject)
		}
		
		// Set as current project
		if err := setCurrentProject(selectedProject); err != nil {
			return fmt.Errorf("failed to set current project: %w", err)
		}
		
		fmt.Printf("Switched to project '%s'\n", selectedProject)
		return nil
	},
}

func listProjects() ([]string, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return nil, err
	}
	
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}
	
	var projects []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "." && entry.Name() != ".." {
			projects = append(projects, entry.Name())
		}
	}
	
	return projects, nil
}

func projectExists(projectName string) bool {
	dataDir, err := getDataDir()
	if err != nil {
		return false
	}
	
	projectPath := filepath.Join(dataDir, projectName)
	info, err := os.Stat(projectPath)
	return err == nil && info.IsDir()
}
