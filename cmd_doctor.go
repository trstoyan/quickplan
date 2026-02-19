package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check project health and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("🏥 Quick Plan Doctor: Checking project '%s'...\n", projectName)

		dataDir, _ := getDataDir()
		projectManager := NewProjectDataManager(dataDir, NewVersionManager(version))

		// 1. Check Lock Status
		fmt.Print("  [1/3] Lock status: ")
		stale, lock, err := projectManager.IsLockStale(projectName)
		if err != nil {
			fmt.Println("✅ No active lock")
		} else if stale {
			fmt.Printf("⚠️  STALE (held by PID %d on %s)\n", lock.PID, lock.Host)
		} else {
			fmt.Printf("🔒 LOCKED (held by PID %d on %s)\n", lock.PID, lock.Host)
		}

		// 2. Check Schema Validity
		fmt.Print("  [2/3] Schema validity: ")
		v11, err := projectManager.LoadProjectV11(projectName)
		if err == nil {
			if err := ValidateProjectV11(v11); err != nil {
				fmt.Printf("❌ Invalid v1.1: %v\n", err)
			} else {
				fmt.Println("✅ Valid v1.1 project.yaml")
			}
		} else if os.IsNotExist(err) {
			// Check legacy
			legacy, err := projectManager.LoadProjectData(projectName)
			if err != nil {
				fmt.Printf("❌ Error loading legacy tasks.yaml: %v\n", err)
			} else {
				fmt.Printf("✅ Valid legacy tasks.yaml (v%s)\n", legacy.Version)
			}
		} else {
			fmt.Printf("❌ Error loading project: %v\n", err)
		}

		// 3. Check Orphan Dependencies
		fmt.Print("  [3/3] Orphan dependencies: ")
		views, _, err := projectManager.GetTaskViews(projectName)
		if err == nil {
			taskIDs := make(map[string]bool)
			for _, v := range views {
				taskIDs[v.ID] = true
			}
			
			orphans := 0
			for _, v := range views {
				for _, dep := range v.DependsOn {
					if !taskIDs[dep] {
						orphans++
					}
				}
			}
			if orphans > 0 {
				fmt.Printf("⚠️  Found %d orphaned dependencies\n", orphans)
			} else {
				fmt.Println("✅ None")
			}
		} else {
			fmt.Println("SKIPPED (load failed)")
		}

		return nil
	},
}

func init() {
	doctorCmd.Flags().StringP("project", "p", "", "Project name")
}
