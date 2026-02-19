package main

import "time"

// ProjectData represents the YAML structure for a project's tasks
type ProjectData struct {
	Version  string     `yaml:"quickplan-cli-version"`
	Tasks    []Task     `yaml:"tasks"`
	Created  time.Time  `yaml:"created"`
	Modified time.Time  `yaml:"modified"`
	Archived bool       `yaml:"archived"`
}

// ProjectConfig represents the project-specific configuration
// This allows projects to sync from different sources (repos, servers)
type ProjectConfig struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description,omitempty"`
	SyncSource  SyncSource `yaml:"sync_source,omitempty"`
	Created     time.Time  `yaml:"created"`
	Modified    time.Time  `yaml:"modified"`
}

// SyncSource defines where the project syncs from
// Following Open/Closed Principle - easy to add new sync types
type SyncSource struct {
	Type   string `yaml:"type,omitempty"`   // "git", "server", "local"
	URL    string `yaml:"url,omitempty"`    // e.g., "team2.quickplan.sh", "git@github.com:..."
	Branch string `yaml:"branch,omitempty"` // For git sources
	Token  string `yaml:"token,omitempty"`  // For authenticated sources (stored separately in secure storage)
}

// NoteEntry represents a single note with its timestamp
type NoteEntry struct {
	Text      string    `yaml:"text"`
	Timestamp time.Time `yaml:"timestamp"`
}

// AgentBehavior defines the "personality" and "loop rules" for an AI agent.
type AgentBehavior struct {
	Role         string `yaml:"role,omitempty"`          // e.g., "Senior Go Architect"
	LifeCycle    string `yaml:"lifecycle,omitempty"`     // e.g., "Atomic" (one-shot) or "Infinite" (loop)
	LoopInterval string `yaml:"loop_interval,omitempty"` // e.g., "30s"
	Strategy     string `yaml:"strategy,omitempty"`      // e.g., "TDD" or "Fast Prototype"
}

// Task represents a single task item
type Task struct {
	ID           int           `yaml:"id"`
	Text         string        `yaml:"text"`
	Done         bool          `yaml:"done"`
	Created      time.Time     `yaml:"created"`
	Completed    *time.Time    `yaml:"completed,omitempty"`
	Notes        []NoteEntry   `yaml:"notes,omitempty"`
	AssignedTo   string        `yaml:"assigned_to,omitempty"`
	DependsOn    []int         `yaml:"depends_on,omitempty"`
	Behavior     AgentBehavior `yaml:"behavior,omitempty"`
	ContextFiles []string      `yaml:"context_files,omitempty"`
	WatchPath    string        `yaml:"watch_path,omitempty"`
}

// Lock represents the lock file metadata
type Lock struct {
	PID       int       `yaml:"pid"`
	Host      string    `yaml:"host"`
	CreatedAt time.Time `yaml:"created_at"`
	TTL       int       `yaml:"ttl_seconds"`
}

// Event represents a single event in the project lifecycle
type Event struct {
	Timestamp  time.Time `yaml:"ts"`
	Type       string    `yaml:"type"`
	Actor      string    `yaml:"actor"`
	TaskID     string    `yaml:"task_id,omitempty"`
	PrevStatus string    `yaml:"prev_status,omitempty"`
	NextStatus string    `yaml:"next_status,omitempty"`
	Message    string    `yaml:"message,omitempty"`
}

// EventLog represents the structure of the events.yaml sidecar
type EventLog struct {
	SchemaVersion string  `yaml:"schema_version"`
	Events        []Event `yaml:"events"`
}
