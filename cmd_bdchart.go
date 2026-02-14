package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var bdchartCmd = &cobra.Command{
	Use:   "bdchart",
	Short: "Display a burndown chart for the current project",
	Long: `Display a text-based burndown chart for the current project,
showing the number of incomplete tasks over time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine target project
		targetProject, err := getCurrentProject()
		if err != nil {
			return fmt.Errorf("failed to get current project: %w", err)
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

		fmt.Println("Generating burndown chart for project:", targetProject)

		burndown := make(map[time.Time]int)

		// Determine project start date
		startDate := projectData.Created.Truncate(24 * time.Hour) // Truncate to get just the date

		// Iterate from start date to today
		for d := startDate; !d.After(time.Now()); d = d.Add(24 * time.Hour) {
			incompleteCount := 0
			for _, task := range projectData.Tasks {
				// Check if task was created on or before the current day
				if !task.Created.Truncate(24 * time.Hour).After(d) {
					// Check if task is not completed, or completed after the current day
					if !task.Done || (task.Completed != nil && task.Completed.Truncate(24*time.Hour).After(d)) {
						incompleteCount++
					}
				}
			}
			burndown[d] = incompleteCount
		}

		// Sort dates for consistent output
		dates := make([]time.Time, 0, len(burndown))
		for d := range burndown {
			dates = append(dates, d)
		}
		// Sort the dates in ascending order
		for i := 0; i < len(dates)-1; i++ {
			for j := i + 1; j < len(dates); j++ {
				if dates[i].After(dates[j]) {
					dates[i], dates[j] = dates[j], dates[i]
				}
			}
		}

		fmt.Println("Burndown Chart for project:", targetProject)
		fmt.Println("Date       | Incomplete Tasks")
		fmt.Println("-----------|-----------------")
		for _, date := range dates {
			count := burndown[date]
			fmt.Printf("%s | %s (%d)\n", date.Format("2006-01-02"), repeatChar('*', count), count)
		}
		
		return nil
	},
}

// Helper function to repeat a character n times
func repeatChar(char rune, count int) string {
	if count <= 0 {
		return ""
	}
	result := make([]rune, count)
	for i := range result {
		result[i] = char
	}
	return string(result)
}

func init() {
	// No specific flags for now, but can add options later (e.g., --days, --start-date)
}
