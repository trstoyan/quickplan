package main

import (
	"fmt"
	"os"
)

func buildStatusIndex(views []TaskView) map[string]string {
	statusByID := make(map[string]string, len(views))
	for _, v := range views {
		statusByID[v.ID] = v.Status
	}
	return statusByID
}

func isTaskRunnable(task TaskView, statusByID map[string]string) bool {
	return taskReadinessIssue(task, statusByID) == ""
}

func taskReadinessIssue(task TaskView, statusByID map[string]string) string {
	if task.Status != "TODO" && task.Status != "PENDING" {
		return fmt.Sprintf("task is not runnable from status %s", task.Status)
	}

	return taskPrerequisiteIssue(task, statusByID)
}

func taskPrerequisiteIssue(task TaskView, statusByID map[string]string) string {
	for _, dep := range task.DependsOn {
		if statusByID[dep] != "DONE" {
			return fmt.Sprintf("dependency %s is not DONE", dep)
		}
	}

	if task.WatchPath != "" {
		if _, err := os.Stat(task.WatchPath); err != nil {
			return fmt.Sprintf("watch path is missing: %s", task.WatchPath)
		}
	}

	for _, p := range task.WatchPaths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			return fmt.Sprintf("watch path is missing: %s", p)
		}
	}

	for _, p := range task.RequiresFiles {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			return fmt.Sprintf("required file is missing: %s", p)
		}
	}

	return ""
}
