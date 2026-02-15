package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo the last deletion",
	Long:  `Restore the tasks that were removed in the most recent delete command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, err := getDataDir()
		if err != nil {
			return fmt.Errorf("failed to get data directory: %w", err)
		}

		undoBackupPath := filepath.Join(dataDir, ".undo_backup.yaml")
		
		// Check if backup exists
		if _, err := os.Stat(undoBackupPath); os.IsNotExist(err) {
			return fmt.Errorf("no undo information found or last action was not a deletion")
		}

		// Read backup
		data, err := os.ReadFile(undoBackupPath)
		if err != nil {
			return fmt.Errorf("failed to read undo backup: %w", err)
		}

		var undoData struct {
			ProjectName string      `yaml:"project_name"`
			Data        ProjectData `yaml:"data"`
		}

		if err := yaml.Unmarshal(data, &undoData); err != nil {
			return fmt.Errorf("failed to parse undo backup: %w", err)
		}

		// Restore project data
		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		if err := projectManager.SaveProjectData(undoData.ProjectName, &undoData.Data); err != nil {
			return fmt.Errorf("failed to restore project data: %w", err)
		}

		// Delete backup file after successful undo
		os.Remove(undoBackupPath)

		fmt.Printf("Successfully restored tasks for project '%s'\n", undoData.ProjectName)
		return nil
	},
}

func init() {
	// Registered in main.go
}
