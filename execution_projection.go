package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	executionProjectionBundleVersion = "execution-projection-bundle/v1"
	defaultExecutionProjectionLimit  = 200
)

type ExecutionProjectionBundle struct {
	BridgeVersion string                          `json:"bridge_version"`
	ProjectName   string                          `json:"project_name"`
	ExportedAt    time.Time                       `json:"exported_at"`
	EventWindow   ExecutionProjectionEventWindow  `json:"event_window"`
	Snapshot      ExecutionProjectionSnapshotJSON `json:"snapshot"`
	Events        []ExecutionProjectionEvent      `json:"events"`
}

type ExecutionProjectionEventWindow struct {
	Limit          int  `json:"limit"`
	TotalEvents    int  `json:"total_events"`
	IncludedEvents int  `json:"included_events"`
	StartSequence  int  `json:"start_sequence"`
	EndSequence    int  `json:"end_sequence"`
	Truncated      bool `json:"truncated"`
}

type ExecutionProjectionSnapshotJSON struct {
	Total       int    `json:"total"`
	Pending     int    `json:"pending"`
	Blocked     int    `json:"blocked"`
	InProgress  int    `json:"in_progress"`
	Retrying    int    `json:"retrying"`
	Done        int    `json:"done"`
	Failed      int    `json:"failed"`
	Cancelled   int    `json:"cancelled"`
	Runnable    int    `json:"runnable"`
	AllTerminal bool   `json:"all_terminal"`
	Summary     string `json:"summary"`
}

type ExecutionProjectionEvent struct {
	Sequence   int       `json:"sequence"`
	Timestamp  time.Time `json:"timestamp"`
	Type       string    `json:"type"`
	Actor      string    `json:"actor"`
	TaskID     string    `json:"task_id,omitempty"`
	PrevStatus string    `json:"prev_status,omitempty"`
	NextStatus string    `json:"next_status,omitempty"`
	Message    string    `json:"message,omitempty"`
}

func (pdm *ProjectDataManager) BuildExecutionProjectionBundle(projectName string, limit int) (*ExecutionProjectionBundle, error) {
	if limit <= 0 {
		limit = defaultExecutionProjectionLimit
	}

	eventLog, err := pdm.loadCanonicalEventLog(projectName)
	if err != nil {
		return nil, err
	}

	snapshot, err := pdm.GetExecutionSnapshot(projectName)
	if err != nil {
		return nil, err
	}

	totalEvents := len(eventLog.Events)
	startIndex := 0
	if totalEvents > limit {
		startIndex = totalEvents - limit
	}

	events := make([]ExecutionProjectionEvent, 0, totalEvents-startIndex)
	for idx, event := range eventLog.Events[startIndex:] {
		events = append(events, ExecutionProjectionEvent{
			Sequence:   startIndex + idx + 1,
			Timestamp:  event.Timestamp,
			Type:       event.Type,
			Actor:      event.Actor,
			TaskID:     event.TaskID,
			PrevStatus: event.PrevStatus,
			NextStatus: event.NextStatus,
			Message:    event.Message,
		})
	}

	window := ExecutionProjectionEventWindow{
		Limit:          limit,
		TotalEvents:    totalEvents,
		IncludedEvents: len(events),
		Truncated:      startIndex > 0,
	}
	if len(events) > 0 {
		window.StartSequence = events[0].Sequence
		window.EndSequence = events[len(events)-1].Sequence
	}

	return &ExecutionProjectionBundle{
		BridgeVersion: executionProjectionBundleVersion,
		ProjectName:   projectName,
		ExportedAt:    time.Now().UTC(),
		EventWindow:   window,
		Snapshot: ExecutionProjectionSnapshotJSON{
			Total:       snapshot.Total,
			Pending:     snapshot.Pending,
			Blocked:     snapshot.Blocked,
			InProgress:  snapshot.InProgress,
			Retrying:    snapshot.Retrying,
			Done:        snapshot.Done,
			Failed:      snapshot.Failed,
			Cancelled:   snapshot.Cancelled,
			Runnable:    snapshot.Runnable,
			AllTerminal: snapshot.AllTerminal,
			Summary:     snapshot.Summary(),
		},
		Events: events,
	}, nil
}

func (pdm *ProjectDataManager) loadCanonicalEventLog(projectName string) (*EventLog, error) {
	projectPath := filepath.Join(pdm.dataDir, projectName)
	v11File := filepath.Join(projectPath, "project.yaml")
	if _, err := os.Stat(v11File); err == nil {
		v11, loadErr := pdm.LoadProjectV11(projectName)
		if loadErr != nil {
			return nil, loadErr
		}
		return &EventLog{
			SchemaVersion: "events-0.1",
			Events:        append([]Event{}, v11.Events...),
		}, nil
	}

	return pdm.LoadEvents(projectName)
}

func WriteExecutionProjectionBundle(outPath string, bundle *ExecutionProjectionBundle) error {
	if bundle == nil {
		return fmt.Errorf("execution projection bundle is required")
	}

	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(outPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(outPath, data, 0644)
}
