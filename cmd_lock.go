package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Manage project locks",
}

var lockStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current lock status",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		stale, lock, err := projectManager.IsLockStale(projectName)
		if err != nil {
			fmt.Printf("No active lock found for project '%s'.\n", projectName)
			return nil
		}

		fmt.Printf("Project: %s\n", projectName)
		fmt.Printf("Status:  ")
		if stale {
			fmt.Println("STALE")
		} else {
			fmt.Println("LOCKED")
		}
		fmt.Printf("Owner PID: %d\n", lock.PID)
		fmt.Printf("Host:      %s\n", lock.Host)
		fmt.Printf("Created:   %s\n", lock.CreatedAt.Format(time.RFC3339))

		expiresAt := lock.CreatedAt.Add(time.Duration(lock.TTL) * time.Second)
		remaining := time.Until(expiresAt)
		if remaining > 0 {
			fmt.Printf("TTL:       %d seconds (expires in %s)\n", lock.TTL, remaining.Round(time.Second))
		} else {
			fmt.Printf("TTL:       %d seconds (EXPIRED)\n", lock.TTL)
		}

		return nil
	},
}

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Remove project lock",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		force, _ := cmd.Flags().GetBool("force")

		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		if !force {
			stale, lock, err := projectManager.IsLockStale(projectName)
			if err != nil {
				fmt.Printf("No lock file found for project '%s'.\n", projectName)
				return nil
			}
			if !stale {
				return fmt.Errorf("refusing to unlock active lock held by PID %d on host %s (use --force to override)", lock.PID, lock.Host)
			}
		}

		if err := projectManager.ReleaseLock(projectName); err != nil {
			return err
		}

		fmt.Printf("Successfully unlocked project '%s'.\n", projectName)
		return nil
	},
}

func init() {
	lockCmd.AddCommand(lockStatusCmd)
	lockStatusCmd.Flags().StringP("project", "p", "", "Project name")

	unlockCmd.Flags().StringP("project", "p", "", "Project name")
	unlockCmd.Flags().BoolP("force", "f", false, "Force unlock even if lock is active")
}
