package main

import (
	"fmt"
	"strings"
)

// ExecutionSnapshot summarizes scheduler-relevant project state.
type ExecutionSnapshot struct {
	Total       int
	Pending     int
	Blocked     int
	InProgress  int
	Retrying    int
	Done        int
	Failed      int
	Cancelled   int
	Runnable    int
	AllTerminal bool
}

func (s ExecutionSnapshot) Summary() string {
	return fmt.Sprintf(
		"total=%d done=%d failed=%d cancelled=%d pending=%d blocked=%d in_progress=%d retrying=%d runnable=%d",
		s.Total, s.Done, s.Failed, s.Cancelled, s.Pending, s.Blocked, s.InProgress, s.Retrying, s.Runnable,
	)
}

// ClaimNextRunnableTask attempts to claim one runnable task for an agent.
// The claim is done through an IN_PROGRESS transition, so transition validation
// and task readiness checks remain centralized in UpdateTaskStatus.
func (pdm *ProjectDataManager) ClaimNextRunnableTask(projectName, agentID string) (*TaskView, error) {
	if _, err := pdm.ReconcileTaskReadiness(projectName, "swarm"); err != nil {
		return nil, err
	}

	views, _, err := pdm.GetTaskViews(projectName)
	if err != nil {
		return nil, err
	}
	statusByID := buildStatusIndex(views)

	for _, view := range views {
		if view.AssignedTo != "" && view.AssignedTo != agentID {
			continue
		}
		if !isTaskRunnable(view, statusByID) {
			continue
		}

		claimErr := pdm.UpdateTaskStatus(projectName, view.ID, "IN_PROGRESS", agentID)
		if claimErr != nil {
			if isClaimConflict(claimErr) {
				continue
			}
			return nil, claimErr
		}

		claimed := view
		claimed.Status = "IN_PROGRESS"
		claimed.AssignedTo = agentID
		return &claimed, nil
	}

	return nil, nil
}

// GetExecutionSnapshot reports aggregate execution state for swarm scheduling.
func (pdm *ProjectDataManager) GetExecutionSnapshot(projectName string) (ExecutionSnapshot, error) {
	if _, err := pdm.ReconcileTaskReadiness(projectName, "swarm"); err != nil {
		return ExecutionSnapshot{}, err
	}

	views, _, err := pdm.GetTaskViews(projectName)
	if err != nil {
		return ExecutionSnapshot{}, err
	}

	snapshot := ExecutionSnapshot{Total: len(views)}
	statusByID := buildStatusIndex(views)

	for _, view := range views {
		switch canonicalStatus(view.Status) {
		case "DONE":
			snapshot.Done++
		case "FAILED":
			snapshot.Failed++
		case "CANCELLED":
			snapshot.Cancelled++
		case "IN_PROGRESS":
			snapshot.InProgress++
		case "RETRYING":
			snapshot.Retrying++
		case "BLOCKED":
			snapshot.Blocked++
		default:
			snapshot.Pending++
		}

		if isTaskRunnable(view, statusByID) {
			snapshot.Runnable++
		}
	}

	snapshot.AllTerminal = snapshot.Pending == 0 && snapshot.Blocked == 0 && snapshot.InProgress == 0 && snapshot.Retrying == 0
	return snapshot, nil
}

func isClaimConflict(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid transition") ||
		strings.Contains(msg, "cannot transition to in_progress") ||
		strings.Contains(msg, "project is locked")
}
