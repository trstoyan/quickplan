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

	taskViews, isV11, err := projectManager.GetTaskViews(targetProject)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}
	
	fmt.Printf("Tasks in project '%s':\n", targetProject)
	// We might need to load metadata for archived status if not in TaskView
	if !isV11 {
		legacy, _ := projectManager.LoadProjectData(targetProject)
		if legacy.Archived {
			fmt.Println("  [ARCHIVED]")
		}
	}
	fmt.Println()
	
	if len(taskViews) == 0 {
		fmt.Println("  No tasks yet. Add one with 'quickplan add <task>'")
		return nil
	}
	
	// Separate incomplete and completed tasks
	var incompleteTasks []TaskView
	var completedTasks []TaskView
	
	for _, task := range taskViews {
		if task.Status == "DONE" {
			completedTasks = append(completedTasks, task)
		} else {
			incompleteTasks = append(incompleteTasks, task)
		}
	}
	
	// Sort completed tasks (placeholder for stable ID sorting or date if available in TaskView)
	
	// Display incomplete tasks
	if len(incompleteTasks) > 0 {
		for _, task := range incompleteTasks {
			fmt.Printf("  %s. [%s] %s\n", task.ID, getStatusIcon(task.Status), task.Text)
		}
	}
	
	// Display completed tasks in a separate block
	if len(completedTasks) > 0 {
		if len(incompleteTasks) > 0 {
			fmt.Println()
		}
		fmt.Println("Completed tasks:")
		for _, task := range completedTasks {
			fmt.Printf("  %s. [✓] %s\n", task.ID, task.Text)
		}
	}
	
	return nil
}

func getStatusIcon(status string) string {
	switch status {
	case "DONE":
		return "✓"
	case "BLOCKED":
		return "B"
	case "IN_PROGRESS":
		return ">"
	default:
		return " "
	}
}

// listAllProjects displays tasks from all projects
func listAllProjects(showAll bool) error {
	dataDir, err := getDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	versionManager := NewVersionManager(version)
	projectManager := NewProjectDataManager(dataDir, versionManager)

	// In list all projects, we probably want to see only active projects by default
	// unless showAll is true (which refers to tasks, but let's use it for projects here too if it makes sense)
	// Actually, let's stick to active projects for --all-projects by default.
	projects, err := projectManager.ListProjects(showAll)
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
