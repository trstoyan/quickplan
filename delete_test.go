package main

import (
	"testing"
)

func TestRenumberTasks(t *testing.T) {
	tasks := []Task{
		{ID: 5, Text: "Task A"},
		{ID: 10, Text: "Task B"},
		{ID: 1, Text: "Task C"},
	}

	renumberTasks(tasks)

	if tasks[0].ID != 1 {
		t.Errorf("expected task 0 ID to be 1, got %d", tasks[0].ID)
	}
	if tasks[1].ID != 2 {
		t.Errorf("expected task 1 ID to be 2, got %d", tasks[1].ID)
	}
	if tasks[2].ID != 3 {
		t.Errorf("expected task 2 ID to be 3, got %d", tasks[2].ID)
	}
}

func TestDeleteLogic(t *testing.T) {
	// This test focuses on the core logic of identifying and removing tasks
	// Similar to what's in cmd_delete.go
	tasks := []Task{
		{ID: 1, Text: "Task 1"},
		{ID: 2, Text: "Task 2"},
		{ID: 3, Text: "Task 3"},
		{ID: 4, Text: "Task 4"},
	}

	idsToDelete := []int{1, 3}
	
	// Find indices to delete
	var indicesToDelete []int
	for _, id := range idsToDelete {
		for i, task := range tasks {
			if task.ID == id {
				indicesToDelete = append(indicesToDelete, i)
				break
			}
		}
	}

	// Sort indices in descending order
	for i := 0; i < len(indicesToDelete); i++ {
		for j := i + 1; j < len(indicesToDelete); j++ {
			if indicesToDelete[i] < indicesToDelete[j] {
				indicesToDelete[i], indicesToDelete[j] = indicesToDelete[j], indicesToDelete[i]
			}
		}
	}

	// Delete tasks
	for _, idx := range indicesToDelete {
		tasks = append(tasks[:idx], tasks[idx+1:]...)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks remaining, got %d", len(tasks))
	}

	if tasks[0].Text != "Task 2" {
		t.Errorf("expected Task 2, got %s", tasks[0].Text)
	}

	if tasks[1].Text != "Task 4" {
		t.Errorf("expected Task 4, got %s", tasks[1].Text)
	}

	// Renumber
	renumberTasks(tasks)

	if tasks[0].ID != 1 || tasks[1].ID != 2 {
		t.Errorf("renumbering failed")
	}
}
