package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// ProjectDataManager handles project data operations
// Following Single Responsibility Principle - manages only project data I/O
type ProjectDataManager struct {
	dataDir        string
	versionManager *VersionManager
}

// NewProjectDataManager creates a new project data manager
func NewProjectDataManager(dataDir string, versionManager *VersionManager) *ProjectDataManager {
	return &ProjectDataManager{
		dataDir:        dataDir,
		versionManager: versionManager,
	}
}

// getLockPath returns the path to the lock file for a project
func (pdm *ProjectDataManager) getLockPath(projectName string) string {
	return filepath.Join(pdm.dataDir, projectName, ".quickplan.lock")
}

// getEventsPath returns the path to the events file for a project
func (pdm *ProjectDataManager) getEventsPath(projectName string) string {
	return filepath.Join(pdm.dataDir, projectName, "events.yaml")
}

// GetTaskStatus returns the status string for a legacy task
func GetTaskStatus(task Task) string {
	if task.Done {
		return "DONE"
	}
	for _, note := range task.Notes {
		if strings.Contains(strings.ToUpper(note.Text), "BLOCKED") {
			return "BLOCKED"
		}
	}
	return "TODO"
}

// AppendEvent appends an event to the project audit trail.
func (pdm *ProjectDataManager) AppendEvent(projectName string, event Event) error {
	projectPath := filepath.Join(pdm.dataDir, projectName)
	v11File := filepath.Join(projectPath, "project.yaml")

	// Try v1.1 embedding first
	if _, err := os.Stat(v11File); err == nil {
		v11, err := pdm.LoadProjectV11(projectName)
		if err != nil {
			return fmt.Errorf("failed to load v1.1 for event append: %w", err)
		}
		v11.Events = append(v11.Events, event)
		return pdm.SaveProjectV11(projectName, v11)
	}

	// Fallback to events.yaml sidecar
	eventsPath := pdm.getEventsPath(projectName)

	// Ensure project directory exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("project '%s' does not exist", projectName)
	}

	// Load existing events or create new log
	var eventLog EventLog
	data, err := os.ReadFile(eventsPath)
	if err == nil {
		if err := yaml.Unmarshal(data, &eventLog); err != nil {
			// If corrupt, we'll just start fresh or return error?
			// Protocol says append-only, so let's try to preserve if possible.
			return fmt.Errorf("failed to parse events file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read events file: %w", err)
	}

	if eventLog.SchemaVersion == "" {
		eventLog.SchemaVersion = "events-0.1"
	}

	eventLog.Events = append(eventLog.Events, event)

	// Marshal and write
	out, err := yaml.Marshal(eventLog)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	return os.WriteFile(eventsPath, out, 0644)
}

// LoadEvents loads the events from the events.yaml sidecar
func (pdm *ProjectDataManager) LoadEvents(projectName string) (*EventLog, error) {
	eventsPath := pdm.getEventsPath(projectName)
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &EventLog{SchemaVersion: "events-0.1", Events: []Event{}}, nil
		}
		return nil, fmt.Errorf("failed to read events file: %w", err)
	}

	var eventLog EventLog
	if err := yaml.Unmarshal(data, &eventLog); err != nil {
		return nil, fmt.Errorf("failed to parse events file: %w", err)
	}

	return &eventLog, nil
}

// AcquireLock attempts to acquire a lock for the project
func (pdm *ProjectDataManager) AcquireLock(projectName string, ttl int) error {
	lockPath := pdm.getLockPath(projectName)

	// Try to create the lock file with O_EXCL
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			// Project directory might not exist yet (e.g. during creation)
			return nil
		}
		if os.IsExist(err) {
			// Lock already exists, check if it's stale
			stale, existingLock, err := pdm.IsLockStale(projectName)
			if err != nil {
				return fmt.Errorf("failed to check if lock is stale: %w", err)
			}
			if stale {
				fmt.Fprintf(os.Stderr, "Warning: stale lock detected from pid %d on host %s, overriding...\n", existingLock.PID, existingLock.Host)
				// Remove stale lock and try again once
				os.Remove(lockPath)
				return pdm.AcquireLock(projectName, ttl)
			}
			return fmt.Errorf("project is locked by pid %d on host %s (created at %s)", existingLock.PID, existingLock.Host, existingLock.CreatedAt.Format(time.RFC3339))
		}
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	defer f.Close()

	host, _ := os.Hostname()
	lock := Lock{
		PID:       os.Getpid(),
		Host:      host,
		CreatedAt: time.Now(),
		TTL:       ttl,
	}

	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("failed to marshal lock data: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write lock data: %w", err)
	}

	return nil
}

// ReleaseLock removes the lock file
func (pdm *ProjectDataManager) ReleaseLock(projectName string) error {
	lockPath := pdm.getLockPath(projectName)
	err := os.Remove(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

// IsLockStale checks if a lock is stale
func (pdm *ProjectDataManager) IsLockStale(projectName string) (bool, *Lock, error) {
	lockPath := pdm.getLockPath(projectName)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return false, nil, err
	}

	var lock Lock
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return true, nil, nil // Corrupt lock is stale
	}

	// 1. Check TTL
	if time.Now().After(lock.CreatedAt.Add(time.Duration(lock.TTL) * time.Second)) {
		return true, &lock, nil
	}

	// 2. Best-effort PID check if on same host
	host, _ := os.Hostname()
	if lock.Host == host {
		process, err := os.FindProcess(lock.PID)
		if err != nil {
			return true, &lock, nil
		}
		// On Unix, FindProcess always succeeds. Use signal 0 to check existence.
		if runtime.GOOS != "windows" {
			err = process.Signal(syscall.Signal(0))
			if err != nil {
				return true, &lock, nil
			}
		}
	}

	return false, &lock, nil
}

// LoadProjectV11 loads a schema v1.1 project file
func (pdm *ProjectDataManager) LoadProjectV11(projectName string) (*ProjectV11, error) {
	projectPath := filepath.Join(pdm.dataDir, projectName)
	v11File := filepath.Join(projectPath, "project.yaml")

	data, err := os.ReadFile(v11File)
	if err != nil {
		return nil, err
	}

	var project ProjectV11
	if err := yaml.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("failed to parse project.yaml: %w", err)
	}

	if project.SchemaVersion != "1.1" {
		return nil, fmt.Errorf("unsupported schema version: %s", project.SchemaVersion)
	}

	return &project, nil
}

// SaveProjectV11 saves a schema v1.1 project file
func (pdm *ProjectDataManager) SaveProjectV11(projectName string, project *ProjectV11) error {
	if err := ValidateProjectV11(project); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := pdm.AcquireLock(projectName, 300); err != nil {
		return err
	}
	defer pdm.ReleaseLock(projectName)

	projectPath := filepath.Join(pdm.dataDir, projectName)
	v11File := filepath.Join(projectPath, "project.yaml")

	project.Project.UpdatedAt = time.Now()

	data, err := yaml.Marshal(project)
	if err != nil {
		return fmt.Errorf("failed to marshal project v1.1: %w", err)
	}

	return os.WriteFile(v11File, data, 0644)
}

// ValidateProjectV11 enforces schema v1.1 rules and invariants.
func ValidateProjectV11(project *ProjectV11) error {
	if project.SchemaVersion != "1.1" {
		return fmt.Errorf("unsupported schema version: %s (expected 1.1)", project.SchemaVersion)
	}

	taskIDs := make(map[string]bool)
	for _, task := range project.Tasks {
		// 1. Unique task ids
		if task.ID == "" {
			return fmt.Errorf("task ID cannot be empty")
		}
		if taskIDs[task.ID] {
			return fmt.Errorf("duplicate task ID: %s", task.ID)
		}
		taskIDs[task.ID] = true

		// 2. Valid status enum
		if !isValidStatus(task.Status) {
			return fmt.Errorf("invalid status for task %s: %s", task.ID, task.Status)
		}
	}

	// 3. depends_on references exist and no cycles
	for _, task := range project.Tasks {
		for _, depID := range task.DependsOn {
			if !taskIDs[depID] {
				return fmt.Errorf("task %s depends on non-existent task %s", task.ID, depID)
			}
		}
	}

	if hasDependencyCycles(project.Tasks) {
		return fmt.Errorf("dependency cycle detected")
	}

	return nil
}

func isValidStatus(status string) bool {
	switch status {
	case "TODO", "PENDING", "BLOCKED", "IN_PROGRESS", "DONE", "FAILED", "RETRYING", "CANCELLED":
		return true
	}
	return false
}

func hasDependencyCycles(tasks []TaskV11) bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	taskMap := make(map[string]TaskV11)

	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	var check func(id string) bool
	check = func(id string) bool {
		visited[id] = true
		recStack[id] = true

		for _, depID := range taskMap[id].DependsOn {
			if !visited[depID] {
				if check(depID) {
					return true
				}
			} else if recStack[depID] {
				return true
			}
		}

		recStack[id] = false
		return false
	}

	for _, task := range tasks {
		if !visited[task.ID] {
			if check(task.ID) {
				return true
			}
		}
	}

	return false
}

// GetTaskViews returns a list of TaskViews for a project, supporting both legacy and v1.1
func (pdm *ProjectDataManager) GetTaskViews(projectName string) ([]TaskView, bool, error) {
	// Try v1.1 first
	v11, err := pdm.LoadProjectV11(projectName)
	if err == nil {
		views := make([]TaskView, len(v11.Tasks))
		for i, t := range v11.Tasks {
			views[i] = TaskView{
				ID:         t.ID,
				Text:       t.Name,
				Status:     t.Status,
				AssignedTo: t.AssignedTo,
				DependsOn:  t.DependsOn,
				Behavior:   t.Behavior,
				IsV11:      true,
			}
		}
		return views, true, nil
	}

	// Fallback to legacy
	legacy, err := pdm.LoadProjectData(projectName)
	if err != nil {
		return nil, false, err
	}

	views := make([]TaskView, len(legacy.Tasks))
	for i, t := range legacy.Tasks {
		// Map int deps to strings
		deps := make([]string, len(t.DependsOn))
		for j, d := range t.DependsOn {
			deps[j] = fmt.Sprintf("t-%d", d)
		}

		views[i] = TaskView{
			ID:         fmt.Sprintf("t-%d", t.ID),
			Text:       t.Text,
			Status:     GetTaskStatus(t),
			AssignedTo: t.AssignedTo,
			DependsOn:  deps,
			WatchPath:  t.WatchPath,
			Behavior:   t.Behavior,
			IsV11:      false,
		}
	}
	return views, false, nil
}

// LoadProjectData loads project data from disk with version migration
func (pdm *ProjectDataManager) LoadProjectData(projectName string) (*ProjectData, error) {
	projectPath := filepath.Join(pdm.dataDir, projectName)
	tasksFile := filepath.Join(projectPath, "tasks.yaml")

	var projectData ProjectData

	// Read existing file if it exists
	data, err := os.ReadFile(tasksFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty project data
			return &ProjectData{
				Version:  pdm.versionManager.currentVersion,
				Tasks:    []Task{},
				Created:  time.Now(),
				Modified: time.Now(),
				Archived: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, &projectData); err != nil {
		return nil, fmt.Errorf("failed to parse tasks file: %w", err)
	}

	// Validate version compatibility
	if err := pdm.versionManager.ValidateProjectVersion(projectData.Version); err != nil {
		return nil, err
	}

	// Migrate if needed
	if projectData.Version != pdm.versionManager.currentVersion {
		migrated, err := pdm.versionManager.MigrateProjectIfNeeded(projectPath)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate project: %w", err)
		}
		if migrated {
			// Reload after migration
			data, err := os.ReadFile(tasksFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read migrated tasks file: %w", err)
			}
			if err := yaml.Unmarshal(data, &projectData); err != nil {
				return nil, fmt.Errorf("failed to parse migrated tasks file: %w", err)
			}
		}
	}

	return &projectData, nil
}

// SaveProjectData saves project data to disk with version tracking
func (pdm *ProjectDataManager) SaveProjectData(projectName string, projectData *ProjectData) error {
	if err := pdm.AcquireLock(projectName, 300); err != nil {
		return err
	}
	defer pdm.ReleaseLock(projectName)

	projectPath := filepath.Join(pdm.dataDir, projectName)
	tasksFile := filepath.Join(projectPath, "tasks.yaml")

	// Ensure version is set
	if projectData.Version == "" {
		projectData.Version = pdm.versionManager.currentVersion
	}

	// Update modified timestamp
	projectData.Modified = time.Now()

	// Marshal to YAML
	data, err := yaml.Marshal(projectData)
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	// Write to file
	if err := os.WriteFile(tasksFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	return nil
}

// CreateProject creates a new project directory and initializes data
func (pdm *ProjectDataManager) CreateProject(projectName string) error {
	projectPath := filepath.Join(pdm.dataDir, projectName)

	// Create project directory
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	now := time.Now()

	// Initialize project configuration
	projectConfig := &ProjectConfig{
		Name:     projectName,
		Created:  now,
		Modified: now,
		SyncSource: SyncSource{
			Type: "local", // Default to local, can be changed later
		},
	}

	// Save project configuration
	if err := pdm.SaveProjectConfig(projectName, projectConfig); err != nil {
		return fmt.Errorf("failed to save project config: %w", err)
	}

	// Initialize project data
	projectData := &ProjectData{
		Version:  pdm.versionManager.currentVersion,
		Tasks:    []Task{},
		Created:  now,
		Modified: now,
		Archived: false,
	}

	// Save initial tasks data
	return pdm.SaveProjectData(projectName, projectData)
}

// LoadProjectConfig loads project configuration from project.yml
func (pdm *ProjectDataManager) LoadProjectConfig(projectName string) (*ProjectConfig, error) {
	projectPath := filepath.Join(pdm.dataDir, projectName)
	configFile := filepath.Join(projectPath, "project.yml")

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist, create default
			return pdm.createDefaultConfig(projectName)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveProjectConfig saves project configuration to project.yml
func (pdm *ProjectDataManager) SaveProjectConfig(projectName string, config *ProjectConfig) error {
	if err := pdm.AcquireLock(projectName, 300); err != nil {
		return err
	}
	defer pdm.ReleaseLock(projectName)

	projectPath := filepath.Join(pdm.dataDir, projectName)
	configFile := filepath.Join(projectPath, "project.yml")

	// Update modified timestamp
	config.Modified = time.Now()

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateTaskStatus updates the status and assigned agent of a specific task.
func (pdm *ProjectDataManager) UpdateTaskStatus(projectName, taskID, status, agentID string) error {
	// Try v1.1 first
	if v11, err := pdm.LoadProjectV11(projectName); err == nil {
		found := false
		for i := range v11.Tasks {
			if v11.Tasks[i].ID == taskID {
				prevStatus := v11.Tasks[i].Status
				v11.Tasks[i].Status = status
				v11.Tasks[i].AssignedTo = agentID
				v11.Tasks[i].UpdatedAt = time.Now()

				v11.Events = append(v11.Events, Event{
					Timestamp:  time.Now(),
					Type:       "TASK_STATUS_CHANGED",
					Actor:      agentID,
					TaskID:     taskID,
					PrevStatus: prevStatus,
					NextStatus: status,
					Message:    fmt.Sprintf("Status updated to %s", status),
				})
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("task %s not found in project %s", taskID, projectName)
		}
		return pdm.SaveProjectV11(projectName, v11)
	}

	// Fallback to legacy
	projectData, err := pdm.LoadProjectData(projectName)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(strings.TrimPrefix(taskID, "t-"))
	if err != nil {
		return fmt.Errorf("invalid legacy task ID: %s", taskID)
	}

	found := false
	for i := range projectData.Tasks {
		if projectData.Tasks[i].ID == id {
			prevStatus := GetTaskStatus(projectData.Tasks[i])
			projectData.Tasks[i].Done = (status == "DONE")
			projectData.Tasks[i].AssignedTo = agentID
			if projectData.Tasks[i].Done {
				now := time.Now()
				projectData.Tasks[i].Completed = &now
			}

			pdm.AppendEvent(projectName, Event{
				Timestamp:  time.Now(),
				Type:       "TASK_STATUS_CHANGED",
				Actor:      agentID,
				TaskID:     taskID,
				PrevStatus: prevStatus,
				NextStatus: status,
				Message:    fmt.Sprintf("Status updated to %s", status),
			})
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("task %s not found in project %s", taskID, projectName)
	}

	return pdm.SaveProjectData(projectName, projectData)
}

// ListProjects returns a list of project names, optionally including archived ones
func (pdm *ProjectDataManager) ListProjects(includeArchived bool) ([]string, error) {
	entries, err := os.ReadDir(pdm.dataDir)
	if err != nil {
		return nil, err
	}

	// Initialize ignore filter
	ignoreFilter := NewIgnoreFilter()
	if err := ignoreFilter.LoadIgnoreFile(pdm.dataDir); err != nil {
		// Log warning but continue with default patterns
		fmt.Fprintf(os.Stderr, "Warning: failed to load ignore file: %v\n", err)
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "." && entry.Name() != ".." {
			// Apply ignore filter
			if !ignoreFilter.ShouldIgnore(entry.Name()) {
				if !includeArchived {
					// Check if project is archived
					projectData, err := pdm.LoadProjectData(entry.Name())
					if err == nil && projectData.Archived {
						continue
					}
				}
				projects = append(projects, entry.Name())
			}
		}
	}

	return projects, nil
}

// createDefaultConfig creates a default project configuration
func (pdm *ProjectDataManager) createDefaultConfig(projectName string) (*ProjectConfig, error) {
	config := &ProjectConfig{
		Name:     projectName,
		Created:  time.Now(),
		Modified: time.Now(),
		SyncSource: SyncSource{
			Type: "local",
		},
	}

	if err := pdm.SaveProjectConfig(projectName, config); err != nil {
		return nil, err
	}

	return config, nil
}
