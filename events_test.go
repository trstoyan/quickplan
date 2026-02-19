package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendEvent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "quickplan-events-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "events-project"
	projectPath := filepath.Join(tmpDir, projectName)
	os.MkdirAll(projectPath, 0755)

	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))

	// 1. Append first event
	event1 := Event{
		Timestamp: time.Now().Round(time.Second),
		Type:      "TASK_CREATED",
		Actor:     "human",
		TaskID:    "t-1",
		Message:   "Initial task",
	}
	err = pdm.AppendEvent(projectName, event1)
	if err != nil {
		t.Fatalf("Failed to append event: %v", err)
	}

	// 2. Append second event
	event2 := Event{
		Timestamp: time.Now().Add(time.Second).Round(time.Second),
		Type:      "TASK_STATUS_CHANGED",
		Actor:     "human",
		TaskID:    "t-1",
		PrevStatus: "PENDING",
		NextStatus: "DONE",
	}
	err = pdm.AppendEvent(projectName, event2)
	if err != nil {
		t.Fatalf("Failed to append second event: %v", err)
	}

	// 3. Load and verify
	eventLog, err := pdm.LoadEvents(projectName)
	if err != nil {
		t.Fatalf("Failed to load events: %v", err)
	}

	if len(eventLog.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(eventLog.Events))
	}

	if eventLog.Events[0].Type != "TASK_CREATED" {
		t.Errorf("Expected first event type TASK_CREATED, got %s", eventLog.Events[0].Type)
	}

	if eventLog.Events[1].NextStatus != "DONE" {
		t.Errorf("Expected second event next status DONE, got %s", eventLog.Events[1].NextStatus)
	}
}

func TestGetTaskStatus(t *testing.T) {
	task1 := Task{Done: true}
	if status := GetTaskStatus(task1); status != "DONE" {
		t.Errorf("Expected DONE, got %s", status)
	}

	task2 := Task{Done: false, Notes: []NoteEntry{{Text: "This is BLOCKED"}}}
	if status := GetTaskStatus(task2); status != "BLOCKED" {
		t.Errorf("Expected BLOCKED, got %s", status)
	}

	task3 := Task{Done: false, Notes: []NoteEntry{{Text: "Regular note"}}}
	if status := GetTaskStatus(task3); status != "PENDING" {
		t.Errorf("Expected PENDING, got %s", status)
	}
}
