package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List all available projects",
	Long: `List all available projects and show which one is currently active.
By default, only active projects are shown. Use --all to see archived projects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		showAll, _ := cmd.Flags().GetBool("all")

		dataDir, err := getDataDir()
		if err != nil {
			return fmt.Errorf("failed to get data directory: %w", err)
		}

		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		projects, err := projectManager.ListProjects(showAll)
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}
		
		if len(projects) == 0 {
			if showAll {
				fmt.Println("No projects found. Create one with 'quickplan create <name>'")
			} else {
				fmt.Println("No active projects found. Use 'quickplan projects --all' to see archived projects or create one with 'quickplan create <name>'")
			}
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

			// Check if archived for display
			archivedSuffix := ""
			if showAll {
				pData, err := projectManager.LoadProjectData(project)
				if err == nil && pData.Archived {
					archivedSuffix = " [ARCHIVED]"
				}
			}

			fmt.Printf("%s %d. %s%s\n", marker, i+1, project, archivedSuffix)
		}
		
		if current != "none" && current != "" {
			fmt.Printf("\n* = current project\n")
		}
		
		return nil
	},
}

func init() {
	projectsCmd.Flags().BoolP("all", "a", false, "Show all projects, including archived ones")
}
