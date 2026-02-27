package main

import (
	"fmt"
	"strings"
	"time"
)

// ScheduleRetryIfAllowed applies retry-policy orchestration for v1.1 tasks.
// It records failure metadata and, if policy allows, transitions:
// FAILED -> RETRYING -> PENDING (after backoff).
func (pdm *ProjectDataManager) ScheduleRetryIfAllowed(projectName, taskID, actorID, failureReason string) (bool, error) {
	v11, err := pdm.LoadProjectV11(projectName)
	if err != nil {
		// Legacy projects do not support retry_policy metadata.
		return false, nil
	}

	actor := strings.TrimSpace(actorID)
	if actor == "" {
		actor = "system:retry"
	}

	var task *TaskV11
	for i := range v11.Tasks {
		if v11.Tasks[i].ID == taskID {
			task = &v11.Tasks[i]
			break
		}
	}
	if task == nil {
		return false, fmt.Errorf("task %s not found in project %s", taskID, projectName)
	}

	if canonicalStatus(task.Status) != "FAILED" {
		return false, nil
	}

	policy := task.RetryPolicy
	if policy == nil || policy.MaxAttempts <= 0 {
		task.LastError = failureReason
		task.UpdatedAt = time.Now()
		if saveErr := pdm.SaveProjectV11(projectName, v11); saveErr != nil {
			return false, saveErr
		}
		return false, nil
	}

	attemptNum := task.Attempts + 1
	task.Attempts = attemptNum
	task.LastError = failureReason
	task.UpdatedAt = time.Now()
	if err := pdm.SaveProjectV11(projectName, v11); err != nil {
		return false, err
	}

	if attemptNum >= policy.MaxAttempts {
		_ = pdm.AppendEvent(projectName, Event{
			Timestamp:  time.Now(),
			Type:       "TASK_RETRY_EXHAUSTED",
			Actor:      actor,
			TaskID:     taskID,
			PrevStatus: "FAILED",
			NextStatus: "FAILED",
			Message:    fmt.Sprintf("Retry budget exhausted (%d/%d)", attemptNum, policy.MaxAttempts),
		})
		return false, nil
	}

	if err := pdm.UpdateTaskStatus(projectName, taskID, "RETRYING", actor); err != nil {
		return false, err
	}

	backoff := retryBackoffDuration(policy, attemptNum)
	_ = pdm.AppendEvent(projectName, Event{
		Timestamp:  time.Now(),
		Type:       "TASK_RETRY_SCHEDULED",
		Actor:      actor,
		TaskID:     taskID,
		PrevStatus: "FAILED",
		NextStatus: "RETRYING",
		Message:    fmt.Sprintf("Retry %d/%d scheduled after %s", attemptNum, policy.MaxAttempts, backoff),
	})

	go func(delay time.Duration) {
		if delay > 0 {
			time.Sleep(delay)
		}
		_ = pdm.UpdateTaskStatus(projectName, taskID, "PENDING", actor)
	}(backoff)

	return true, nil
}

func retryBackoffDuration(policy *RetryPolicy, attemptNum int) time.Duration {
	if policy == nil {
		return 0
	}
	base := policy.BaseSeconds
	if base < 0 {
		base = 0
	}
	if attemptNum < 1 {
		attemptNum = 1
	}

	switch strings.ToLower(policy.Backoff) {
	case "linear":
		return time.Duration(base*attemptNum) * time.Second
	case "exponential":
		multiplier := 1 << (attemptNum - 1)
		return time.Duration(base*multiplier) * time.Second
	default:
		return time.Duration(base) * time.Second
	}
}
