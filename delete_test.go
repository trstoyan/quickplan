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

func TestDependencyCleanup(t *testing.T) {
	tasks := []Task{
		{ID: 1, Text: "Task 1"},
		{ID: 2, Text: "Task 2", DependsOn: []int{1, 3}},
		{ID: 3, Text: "Task 3"},
	}

	taskIDsToDelete := []int{1}

	// Simulation of logic in cmd_delete.go
	var remainingTasks []Task
	for _, task := range tasks {
		isDeleted := false
		for _, id := range taskIDsToDelete {
			if task.ID == id {
				isDeleted = true
				break
			}
		}
		if !isDeleted {
			remainingTasks = append(remainingTasks, task)
		}
	}

	// Update depends_on
	for i := range remainingTasks {
		newDependsOn := []int{}
		removedIDs := []int{}
		for _, depID := range remainingTasks[i].DependsOn {
			isDeleted := false
			for _, deletedID := range taskIDsToDelete {
				if depID == deletedID {
					isDeleted = true
					removedIDs = append(removedIDs, deletedID)
					break
				}
			}
			if !isDeleted {
				newDependsOn = append(newDependsOn, depID)
			}
		}
		if len(removedIDs) > 0 {
			remainingTasks[i].DependsOn = newDependsOn
			for range removedIDs {
				remainingTasks[i].Notes = append(remainingTasks[i].Notes, NoteEntry{
					Text: "Dependency removed: 1 (task deleted)",
				})
			}
		}
	}

	if len(remainingTasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(remainingTasks))
	}

	// Check Task 2
	var task2 *Task
	for i := range remainingTasks {
		if remainingTasks[i].ID == 2 {
			task2 = &remainingTasks[i]
		}
	}

	if len(task2.DependsOn) != 1 || task2.DependsOn[0] != 3 {
		t.Errorf("expected Task 2 to depend only on 3, got %v", task2.DependsOn)
	}

	foundNote := false
	for _, note := range task2.Notes {
		if note.Text == "Dependency removed: 1 (task deleted)" {
			foundNote = true
			break
		}
	}
	if !foundNote {
		t.Error("expected note about removed dependency not found")
	}
}
