package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate projects between schema versions",
}

var migrateV11Cmd = &cobra.Command{
	Use:   "v1.1",
	Short: "Migrate current project to schema v1.1",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		// Check if already migrated
		v11File := filepath.Join(dataDir, projectName, "project.yaml")
		if _, err := os.Stat(v11File); err == nil && !force {
			return fmt.Errorf("project '%s' is already v1.1 (use --force to overwrite)", projectName)
		}

		// Load legacy data
		legacyData, err := projectManager.LoadProjectData(projectName)
		if err != nil {
			return fmt.Errorf("failed to load legacy data: %w", err)
		}

		legacyConfig, err := projectManager.LoadProjectConfig(projectName)
		if err != nil {
			// Not critical, we can use defaults
			legacyConfig = &ProjectConfig{Name: projectName}
		}

		legacyEvents, err := projectManager.LoadEvents(projectName)
		if err != nil {
			legacyEvents = &EventLog{}
		}

		// Map to v1.1
		v11 := ProjectV11{
			SchemaVersion: "1.1",
			Project: ProjectMeta{
				Name:      legacyConfig.Name,
				Version:   "0.1.0",
				CreatedAt: legacyData.Created,
				UpdatedAt: time.Now(),
			},
			Lock: LockConfig{
				File:       ".quickplan.lock",
				TTLSeconds: 300,
			},
			Tasks:  make([]TaskV11, len(legacyData.Tasks)),
			Events: legacyEvents.Events,
		}

		now := time.Now()

		for i, t := range legacyData.Tasks {
			status := GetTaskStatus(t)
			
			deps := make([]string, len(t.DependsOn))
			for j, d := range t.DependsOn {
				deps[j] = fmt.Sprintf("t-%d", d)
			}

			watch := WatchConfig{}
			if t.WatchPath != "" {
				watch.Paths = []string{t.WatchPath}
			}

			attempts := 0
			if t.Done {
				attempts = 1
			}

			v11.Tasks[i] = TaskV11{
				ID:         fmt.Sprintf("t-%d", t.ID),
				Name:       t.Text,
				Status:     status,
				AssignedTo: t.AssignedTo,
				DependsOn:  deps,
				Watch:      watch,
				Behavior:   t.Behavior,
				Attempts:   attempts,
				UpdatedAt:  now,
			}

			// If no events existed, create basic audit trail
			if len(legacyEvents.Events) == 0 {
				v11.Events = append(v11.Events, Event{
					Timestamp: t.Created,
					Type:      "TASK_CREATED",
					Actor:     "human",
					TaskID:    fmt.Sprintf("t-%d", t.ID),
					Message:   "Imported during migration",
				})
				v11.Events = append(v11.Events, Event{
					Timestamp:  now,
					Type:       "TASK_STATUS_CHANGED",
					Actor:      "system",
					TaskID:     fmt.Sprintf("t-%d", t.ID),
					NextStatus: status,
					Message:    "Migration snapshot",
				})
			}
		}

		if dryRun {
			out, _ := yaml.Marshal(v11)
			fmt.Println("--- DRY RUN: project.yaml ---")
			fmt.Println(string(out))
			return nil
		}

		// Atomic write
		tmpFile := v11File + ".tmp"
		out, err := yaml.Marshal(v11)
		if err != nil {
			return err
		}
		if err := os.WriteFile(tmpFile, out, 0644); err != nil {
			return err
		}
		if err := os.Rename(tmpFile, v11File); err != nil {
			return err
		}

		fmt.Printf("Successfully migrated project '%s' to schema v1.1\n", projectName)
		fmt.Println("Note: Legacy files (tasks.yaml, project.yml, events.yaml) were preserved.")
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(migrateV11Cmd)
	migrateV11Cmd.Flags().StringP("project", "p", "", "Project to migrate")
	migrateV11Cmd.Flags().Bool("dry-run", false, "Preview migration without writing")
	migrateV11Cmd.Flags().Bool("force", false, "Overwrite existing project.yaml")
}
