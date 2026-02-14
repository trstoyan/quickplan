package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks in the current or specified project",
	Long: `List all tasks in the current project, or in a specified project
using the --project flag. Shows task ID, text, and completion status.
Use --all-projects to list tasks from all projects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		showAll, _ := cmd.Flags().GetBool("all")
		allProjects, _ := cmd.Flags().GetBool("all-projects")
		
		if allProjects {
			return listAllProjects(showAll)
		}
		
		// Determine target project
		var targetProject string
		projectFlag, _ := cmd.Flags().GetString("project")
		if projectFlag != "" {
			targetProject = projectFlag
		} else {
			var err error
			targetProject, err = getCurrentProject()
			if err != nil {
				return fmt.Errorf("failed to get current project: %w", err)
			}
		}
		
		return listProjectTasks(targetProject, showAll)
	},
}

func init() {
	listCmd.Flags().StringP("project", "p", "", "List tasks from this project instead of current")
	listCmd.Flags().BoolP("all", "a", false, "Show all tasks including completed ones")
	listCmd.Flags().Bool("all-projects", false, "List tasks from all projects")
}

// listProjectTasks displays tasks for a single project
func listProjectTasks(targetProject string, showAll bool) error {
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
	
	fmt.Printf("Tasks in project '%s':\n", targetProject)
	if projectData.Archived {
		fmt.Println("  [ARCHIVED]")
	}
	fmt.Println()
	
	if len(projectData.Tasks) == 0 {
		fmt.Println("  No tasks yet. Add one with 'quickplan add <task>'")
		return nil
	}
	
	// Separate incomplete and completed tasks
	var incompleteTasks []Task
	var completedTasks []Task
	
	for _, task := range projectData.Tasks {
		if task.Done {
			completedTasks = append(completedTasks, task)
		} else {
			incompleteTasks = append(incompleteTasks, task)
		}
	}
	
	// Sort completed tasks by completion date (latest first)
	sort.Slice(completedTasks, func(i, j int) bool {
		if completedTasks[i].Completed == nil {
			return false
		}
		if completedTasks[j].Completed == nil {
			return true
		}
		return completedTasks[i].Completed.After(*completedTasks[j].Completed)
	})
	
	// Limit completed tasks to latest 5 if not showing all
	var displayCompletedTasks []Task
	if showAll {
		displayCompletedTasks = completedTasks
	} else {
		maxCompleted := 5
		if len(completedTasks) > maxCompleted {
			displayCompletedTasks = completedTasks[:maxCompleted]
		} else {
			displayCompletedTasks = completedTasks
		}
	}
	
	// Display incomplete tasks
	if len(incompleteTasks) > 0 {
		for _, task := range incompleteTasks {
			fmt.Printf("  %d. [ ] %s\n", task.ID, task.Text)
		}
	}
	
	// Display completed tasks in a separate block
	if len(displayCompletedTasks) > 0 {
		if len(incompleteTasks) > 0 {
			fmt.Println()
		}
		fmt.Println("Completed tasks:")
		for _, task := range displayCompletedTasks {
			completedDate := "unknown date"
			if task.Completed != nil {
				completedDate = task.Completed.Format("2006-01-02")
			}
			fmt.Printf("  %d. [✓] %s (completed: %s)\n", task.ID, task.Text, completedDate)
		}
		
		// Show message if there are more completed tasks
		if !showAll && len(completedTasks) > 5 {
			remaining := len(completedTasks) - 5
			fmt.Printf("  ... and %d more completed task(s). Use --all to see all.\n", remaining)
		}
	}
	
	// Show message if no tasks at all
	if len(incompleteTasks) == 0 && len(displayCompletedTasks) == 0 {
		fmt.Println("  No tasks yet. Add one with 'quickplan add <task>'")
	}
	
	return nil
}

// listAllProjects displays tasks from all projects
func listAllProjects(showAll bool) error {
	projects, err := listProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}
	
	if len(projects) == 0 {
		fmt.Println("No projects found. Create one with 'quickplan create <name>'")
		return nil
	}
	
	// Sort projects alphabetically
	sort.Strings(projects)
	
	hasAnyTasks := false
	dataDir, _ := getDataDir()
	versionManager := NewVersionManager(version)
	projectManager := NewProjectDataManager(dataDir, versionManager)

	for i, project := range projects {
		if i > 0 {
			fmt.Println()
		}

		// Check if this project has tasks before displaying
		if projectData, err := projectManager.LoadProjectData(project); err == nil {
			if len(projectData.Tasks) > 0 {
				hasAnyTasks = true
			}
		}

		err := listProjectTasks(project, showAll)
		if err != nil {
			// Skip projects that can't be read, but continue with others
			fmt.Printf("Error loading project '%s': %v\n", project, err)
			continue
		}
	}
	
	if !hasAnyTasks {
		fmt.Println("\nNo tasks found in any project.")
	}
	
	return nil
}
