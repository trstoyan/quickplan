package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildExecutionProjectionBundle_LegacyBoundedWindow(t *testing.T) {
	tmpDir := t.TempDir()
	projectName := "bridge-legacy"
	projectPath := filepath.Join(tmpDir, projectName)
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatal(err)
	}

	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))
	projectData := &ProjectData{
		Version: "0.1.0",
		Tasks: []Task{
			{ID: 1, Text: "done", Done: true, Status: "DONE", Created: time.Now()},
			{ID: 2, Text: "todo", Status: "TODO", Created: time.Now()},
		},
		Created:  time.Now(),
		Modified: time.Now(),
	}
	if err := pdm.SaveProjectData(projectName, projectData); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	events := []Event{
		{Timestamp: time.Unix(100, 0).UTC(), Type: "TASK_CREATED", Actor: "human", TaskID: "t-1", NextStatus: "TODO"},
		{Timestamp: time.Unix(101, 0).UTC(), Type: "TASK_STATUS_CHANGED", Actor: "human", TaskID: "t-1", PrevStatus: "TODO", NextStatus: "DONE"},
		{Timestamp: time.Unix(102, 0).UTC(), Type: "TASK_CREATED", Actor: "human", TaskID: "t-2", NextStatus: "TODO"},
	}
	for _, event := range events {
		if err := pdm.AppendEvent(projectName, event); err != nil {
			t.Fatalf("append event failed: %v", err)
		}
	}

	bundle, err := pdm.BuildExecutionProjectionBundle(projectName, 2)
	if err != nil {
		t.Fatalf("build bundle failed: %v", err)
	}

	if bundle.BridgeVersion != executionProjectionBundleVersion {
		t.Fatalf("unexpected bridge version: %s", bundle.BridgeVersion)
	}
	if bundle.EventWindow.TotalEvents != 3 || bundle.EventWindow.IncludedEvents != 2 {
		t.Fatalf("unexpected event window: %+v", bundle.EventWindow)
	}
	if bundle.EventWindow.StartSequence != 2 || bundle.EventWindow.EndSequence != 3 {
		t.Fatalf("unexpected event sequence window: %+v", bundle.EventWindow)
	}
	if !bundle.EventWindow.Truncated {
		t.Fatal("expected truncated event window")
	}
	if len(bundle.Events) != 2 || bundle.Events[0].Sequence != 2 || bundle.Events[1].Sequence != 3 {
		t.Fatalf("unexpected events: %+v", bundle.Events)
	}
	if bundle.Snapshot.Done != 1 || bundle.Snapshot.Pending != 1 || bundle.Snapshot.Total != 2 {
		t.Fatalf("unexpected snapshot: %+v", bundle.Snapshot)
	}
}

func TestBuildExecutionProjectionBundle_UsesEmbeddedV11Events(t *testing.T) {
	tmpDir := t.TempDir()
	projectName := "bridge-v11"
	pdm := NewProjectDataManager(tmpDir, NewVersionManager("0.1.0"))

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
			{ID: "t-1", Name: "ship", Status: "IN_PROGRESS", UpdatedAt: time.Now()},
		},
		Events: []Event{
			{Timestamp: time.Unix(200, 0).UTC(), Type: "TASK_STATUS_CHANGED", Actor: "worker-1", TaskID: "t-1", PrevStatus: "PENDING", NextStatus: "IN_PROGRESS"},
		},
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, projectName), 0755); err != nil {
		t.Fatal(err)
	}
	if err := pdm.SaveProjectV11(projectName, v11); err != nil {
		t.Fatalf("save v11 failed: %v", err)
	}

	bundle, err := pdm.BuildExecutionProjectionBundle(projectName, 10)
	if err != nil {
		t.Fatalf("build bundle failed: %v", err)
	}

	if len(bundle.Events) != 1 {
		t.Fatalf("expected one event, got %d", len(bundle.Events))
	}
	if bundle.Events[0].Type != "TASK_STATUS_CHANGED" || bundle.Events[0].Sequence != 1 {
		t.Fatalf("unexpected event payload: %+v", bundle.Events[0])
	}
	if bundle.Snapshot.InProgress != 1 || bundle.Snapshot.Total != 1 {
		t.Fatalf("unexpected snapshot: %+v", bundle.Snapshot)
	}
}

func TestWriteExecutionProjectionBundle_WritesJSONFile(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "nested", "bundle.json")
	bundle := &ExecutionProjectionBundle{
		BridgeVersion: executionProjectionBundleVersion,
		ProjectName:   "bridge-write",
		ExportedAt:    time.Unix(300, 0).UTC(),
		EventWindow: ExecutionProjectionEventWindow{
			Limit:          10,
			TotalEvents:    1,
			IncludedEvents: 1,
			StartSequence:  1,
			EndSequence:    1,
		},
		Snapshot: ExecutionProjectionSnapshotJSON{Total: 1, Pending: 1, Summary: "total=1"},
		Events: []ExecutionProjectionEvent{
			{Sequence: 1, Timestamp: time.Unix(300, 0).UTC(), Type: "TASK_CREATED", Actor: "human", TaskID: "t-1"},
		},
	}

	if err := WriteExecutionProjectionBundle(outPath, bundle); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var decoded ExecutionProjectionBundle
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ProjectName != "bridge-write" || len(decoded.Events) != 1 {
		t.Fatalf("unexpected decoded bundle: %+v", decoded)
	}
}
