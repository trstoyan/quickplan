package main

import (
	"testing"
	"time"
)

func TestScheduleRetryIfAllowed_SchedulesRetry(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	v11 := &ProjectV11{
		SchemaVersion: "1.1",
		Project: ProjectMeta{
			Name:      projectName,
			Version:   "0.3.0-alpha.rc1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Lock: LockConfig{
			File:       ".quickplan.lock",
			TTLSeconds: 300,
		},
		Tasks: []TaskV11{
			{
				ID:     "t-1",
				Name:   "retryable",
				Status: "TODO",
				RetryPolicy: &RetryPolicy{
					MaxAttempts: 2,
					Backoff:     "fixed",
					BaseSeconds: 0,
				},
				UpdatedAt: time.Now(),
			},
		},
		Events: []Event{},
	}
	if err := pdm.SaveProjectV11(projectName, v11); err != nil {
		t.Fatalf("save v1.1 failed: %v", err)
	}

	if err := pdm.UpdateTaskStatus(projectName, "t-1", "IN_PROGRESS", "agent-1"); err != nil {
		t.Fatalf("IN_PROGRESS failed: %v", err)
	}
	if err := pdm.UpdateTaskStatus(projectName, "t-1", "FAILED", "agent-1"); err != nil {
		t.Fatalf("FAILED failed: %v", err)
	}

	scheduled, err := pdm.ScheduleRetryIfAllowed(projectName, "t-1", "agent-1", "boom")
	if err != nil {
		t.Fatalf("schedule retry failed: %v", err)
	}
	if !scheduled {
		t.Fatal("expected retry to be scheduled")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		reloaded, err := pdm.LoadProjectV11(projectName)
		if err != nil {
			t.Fatalf("reload failed: %v", err)
		}
		if reloaded.Tasks[0].Status == "PENDING" {
			if reloaded.Tasks[0].Attempts != 1 {
				t.Fatalf("expected attempts=1, got %d", reloaded.Tasks[0].Attempts)
			}
			if reloaded.Tasks[0].LastError != "boom" {
				t.Fatalf("expected last_error=boom, got %s", reloaded.Tasks[0].LastError)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("task did not return to PENDING after scheduled retry")
}

func TestScheduleRetryIfAllowed_ExhaustsBudget(t *testing.T) {
	pdm, projectName, cleanup := newTransitionTestManager(t)
	defer cleanup()

	v11 := &ProjectV11{
		SchemaVersion: "1.1",
		Project: ProjectMeta{
			Name:      projectName,
			Version:   "0.3.0-alpha.rc1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Lock: LockConfig{
			File:       ".quickplan.lock",
			TTLSeconds: 300,
		},
		Tasks: []TaskV11{
			{
				ID:     "t-1",
				Name:   "single-attempt",
				Status: "TODO",
				RetryPolicy: &RetryPolicy{
					MaxAttempts: 1,
					Backoff:     "fixed",
					BaseSeconds: 0,
				},
				UpdatedAt: time.Now(),
			},
		},
		Events: []Event{},
	}
	if err := pdm.SaveProjectV11(projectName, v11); err != nil {
		t.Fatalf("save v1.1 failed: %v", err)
	}

	if err := pdm.UpdateTaskStatus(projectName, "t-1", "IN_PROGRESS", "agent-1"); err != nil {
		t.Fatalf("IN_PROGRESS failed: %v", err)
	}
	if err := pdm.UpdateTaskStatus(projectName, "t-1", "FAILED", "agent-1"); err != nil {
		t.Fatalf("FAILED failed: %v", err)
	}

	scheduled, err := pdm.ScheduleRetryIfAllowed(projectName, "t-1", "agent-1", "boom")
	if err != nil {
		t.Fatalf("schedule retry failed: %v", err)
	}
	if scheduled {
		t.Fatal("expected retry budget to be exhausted")
	}

	reloaded, err := pdm.LoadProjectV11(projectName)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.Tasks[0].Status != "FAILED" {
		t.Fatalf("expected FAILED status, got %s", reloaded.Tasks[0].Status)
	}
	if reloaded.Tasks[0].Attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", reloaded.Tasks[0].Attempts)
	}
}

func TestRetryBackoffDuration(t *testing.T) {
	p := &RetryPolicy{Backoff: "linear", BaseSeconds: 2}
	if got := retryBackoffDuration(p, 3); got != 6*time.Second {
		t.Fatalf("linear backoff mismatch: %s", got)
	}

	p.Backoff = "exponential"
	if got := retryBackoffDuration(p, 3); got != 8*time.Second {
		t.Fatalf("exponential backoff mismatch: %s", got)
	}

	p.Backoff = "fixed"
	if got := retryBackoffDuration(p, 3); got != 2*time.Second {
		t.Fatalf("fixed backoff mismatch: %s", got)
	}
}
