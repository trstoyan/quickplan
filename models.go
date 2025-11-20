package main

import "time"

// ProjectData represents the YAML structure for a project
type ProjectData struct {
	Tasks    []Task     `yaml:"tasks"`
	Created  time.Time  `yaml:"created"`
	Modified time.Time  `yaml:"modified"`
	Archived bool       `yaml:"archived"`
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
