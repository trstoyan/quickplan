package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List all available projects",
	Long: `List all available projects and show which one is currently active.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := listProjects()
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}
		
		if len(projects) == 0 {
			fmt.Println("No projects found. Create one with 'quickplan create <name>'")
			return nil
		}
		
		// Get current project
		current, err := getCurrentProject()
		if err != nil {
			current = "none"
		}
		
		fmt.Printf("Available projects:\n\n")
		for i, project := range projects {
			marker := " "
			if project == current {
				marker = "*"
			}
			fmt.Printf("%s %d. %s\n", marker, i+1, project)
		}
		
		if current != "none" && current != "" {
			fmt.Printf("\n* = current project\n")
		}
		
		return nil
	},
}

// init function not needed - command registered in main.go
