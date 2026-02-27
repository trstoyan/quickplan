package main

import (
	"strings"
	"time"
)

// ReconcileTaskReadiness aligns task status with dependency and guard readiness.
// - TODO/PENDING tasks with unmet prerequisites are moved to BLOCKED.
// - BLOCKED tasks with all prerequisites satisfied are moved to PENDING.
func (pdm *ProjectDataManager) ReconcileTaskReadiness(projectName, actorID string) (int, error) {
	views, _, err := pdm.GetTaskViews(projectName)
	if err != nil {
		return 0, err
	}

	actor := strings.TrimSpace(actorID)
	if actor == "" {
		actor = "system:guard"
	}

	statusByID := buildStatusIndex(views)
	changed := 0

	for _, task := range views {
		current := canonicalStatus(task.Status)
		if current != "PENDING" && current != "BLOCKED" {
			continue
		}

		issue := taskPrerequisiteIssue(task, statusByID)

		if issue != "" && current != "BLOCKED" {
			if err := pdm.UpdateTaskStatus(projectName, task.ID, "BLOCKED", ""); err != nil {
				return changed, err
			}

			_ = pdm.AppendEvent(projectName, Event{
				Timestamp:  time.Now(),
				Type:       "TASK_BLOCKED",
				Actor:      actor,
				TaskID:     task.ID,
				PrevStatus: task.Status,
				NextStatus: "BLOCKED",
				Message:    issue,
			})

			statusByID[task.ID] = "BLOCKED"
			changed++
			continue
		}

		if issue == "" && current == "BLOCKED" {
			if err := pdm.UpdateTaskStatus(projectName, task.ID, "PENDING", ""); err != nil {
				return changed, err
			}

			_ = pdm.AppendEvent(projectName, Event{
				Timestamp:  time.Now(),
				Type:       "TASK_UNBLOCKED",
				Actor:      actor,
				TaskID:     task.ID,
				PrevStatus: task.Status,
				NextStatus: "PENDING",
				Message:    "All dependencies and guard checks are satisfied",
			})

			statusByID[task.ID] = "PENDING"
			changed++
		}
	}

	return changed, nil
}
