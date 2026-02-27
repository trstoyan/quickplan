package main

import (
	"fmt"
	"strings"
)

func canonicalStatus(status string) string {
	s := strings.ToUpper(strings.TrimSpace(status))
	if s == "TODO" {
		return "PENDING"
	}
	return s
}

func isAllowedTransition(current, next string) bool {
	if current == next {
		return true
	}
	if next == "CANCELLED" {
		return true
	}

	allowed := map[string]map[string]bool{
		"PENDING": {
			"IN_PROGRESS": true,
			"BLOCKED":     true,
		},
		"IN_PROGRESS": {
			"DONE":   true,
			"FAILED": true,
		},
		"FAILED": {
			"RETRYING": true,
		},
		"RETRYING": {
			"PENDING": true,
		},
		"BLOCKED": {
			"PENDING": true,
		},
	}

	return allowed[current][next]
}

func validateTaskStatusTransition(task TaskView, nextStatus string, statusByID map[string]string) error {
	next := canonicalStatus(nextStatus)
	current := canonicalStatus(task.Status)

	if !isValidStatus(nextStatus) {
		return fmt.Errorf("invalid target status: %s", nextStatus)
	}

	if current == "" {
		current = "PENDING"
	}

	if !isAllowedTransition(current, next) {
		return fmt.Errorf("invalid transition: %s -> %s", task.Status, nextStatus)
	}

	if next == "IN_PROGRESS" {
		if issue := taskReadinessIssue(task, statusByID); issue != "" {
			return fmt.Errorf("cannot transition to IN_PROGRESS: %s", issue)
		}
	}

	return nil
}
