package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	rootCmd = &cobra.Command{
		Use:   "quickplan",
		Short: "A fast CLI task manager with project support",
		Long: `QuickPlan is a terminal-based task manager that lets you organize
tasks into named projects with vim-inspired selection menus.`,
		Version: version,
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(changeCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(completeCmd)
	rootCmd.AddCommand(archiveCmd)
}

// Get the data directory for storing projects and tasks
func getDataDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	
	dataDir := filepath.Join(usr.HomeDir, ".local", "share", "quickplan")
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}
	
	return dataDir, nil
}

// Get the current project context
func getCurrentProject() (string, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return "", err
	}
	
	contextFile := filepath.Join(dataDir, ".current_project")
	project, err := os.ReadFile(contextFile)
	if err != nil {
		return "default", nil // Return default if no context file exists
	}
	
	return string(project), nil
}

// Set the current project context
func setCurrentProject(project string) error {
	dataDir, err := getDataDir()
	if err != nil {
		return err
	}
	
	contextFile := filepath.Join(dataDir, ".current_project")
	return os.WriteFile(contextFile, []byte(project), 0644)
}
