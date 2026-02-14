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

// Task represents a single task item
type Task struct {
	ID        int        `yaml:"id"`
	Text      string     `yaml:"text"`
	Done      bool       `yaml:"done"`
	Created   time.Time  `yaml:"created"`
	Completed *time.Time `yaml:"completed,omitempty"`
	Notes     []NoteEntry   `yaml:"notes,omitempty"`
}
