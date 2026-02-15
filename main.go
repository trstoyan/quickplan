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
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(bdchartCmd)
	rootCmd.AddCommand(undoCmd)
}

// Get the data directory for storing projects and tasks
func getDataDir() (string, error) {
	// 1. Allow overriding data directory via environment variable
	if envDir := os.Getenv("QUICKPLAN_DATADIR"); envDir != "" {
		if err := os.MkdirAll(envDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create custom data directory: %w", err)
		}
		return envDir, nil
	}

	// 2. Try standard location (~/.local/share/quickplan)
	usr, err := user.Current()
	if err == nil {
		dataDir := filepath.Join(usr.HomeDir, ".local", "share", "quickplan")
		if err := os.MkdirAll(dataDir, 0755); err == nil {
			ensureIgnoreFile(dataDir)
			return dataDir, nil
		}
	}

	// 3. Fallback to a temporary directory if standard location is unavailable
	tmpDataDir := filepath.Join(os.TempDir(), "quickplan")
	if err := os.MkdirAll(tmpDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create fallback data directory: %w", err)
	}
	
	ensureIgnoreFile(tmpDataDir)
	return tmpDataDir, nil
}

// ensureIgnoreFile ensures a default ignore file exists in the given directory
func ensureIgnoreFile(dataDir string) {
	if err := CreateDefaultIgnoreFile(dataDir); err != nil {
		// Log warning but continue
		fmt.Fprintf(os.Stderr, "Warning: failed to create default ignore file: %v\n", err)
	}
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
