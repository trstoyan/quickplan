package main

import (
	"testing"
)

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

	if tasks[0].Text != "Task 2" || tasks[0].ID != 2 {
		t.Errorf("expected Task 2 with ID 2, got %s with ID %d", tasks[0].Text, tasks[0].ID)
	}

	if tasks[1].Text != "Task 4" || tasks[1].ID != 4 {
		t.Errorf("expected Task 4 with ID 4, got %s with ID %d", tasks[1].Text, tasks[1].ID)
	}
}
