package main

import "time"

// ProjectV11 represents the Schema v1.1 project structure
type ProjectV11 struct {
	SchemaVersion string      `yaml:"schema_version"`
	Project       ProjectMeta `yaml:"project"`
	Lock          LockConfig  `yaml:"lock"`
	Agents        []AgentMeta `yaml:"agents,omitempty"`
	Tasks         []TaskV11   `yaml:"tasks"`
	Events        []Event     `yaml:"events"`
	Registry      *RegistryConfig `yaml:"registry,omitempty"`
}

type ProjectMeta struct {
	Name      string    `yaml:"name"`
	Version   string    `yaml:"version"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

type LockConfig struct {
	File       string `yaml:"file"`
	TTLSeconds int    `yaml:"ttl_seconds"`
}

type AgentMeta struct {
	ID              string        `yaml:"id"`
	Kind            string        `yaml:"kind"`
	DisplayName     string        `yaml:"display_name"`
	Capabilities    []string      `yaml:"capabilities"`
	DefaultBehavior AgentBehavior `yaml:"default_behavior"`
}

type TaskV11 struct {
	ID          string        `yaml:"id"`
	Name        string        `yaml:"name"`
	Status      string        `yaml:"status"`
	AssignedTo  string        `yaml:"assigned_to,omitempty"`
	DependsOn   []string      `yaml:"depends_on,omitempty"`
	Watch       WatchConfig   `yaml:"watch,omitempty"`
	Behavior    AgentBehavior `yaml:"behavior,omitempty"`
	RetryPolicy *RetryPolicy  `yaml:"retry_policy,omitempty"`
	Attempts    int           `yaml:"attempts"`
	LastError   string        `yaml:"last_error,omitempty"`
	UpdatedAt   time.Time     `yaml:"updated_at"`
}

type WatchConfig struct {
	Paths         []string `yaml:"paths,omitempty"`
	RequiresFiles []string `yaml:"requires_files,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts int    `yaml:"max_attempts"`
	Backoff     string `yaml:"backoff"`
	BaseSeconds int    `yaml:"base_seconds"`
}

type RegistryConfig struct {
	Endpoint  string `yaml:"endpoint"`
	Namespace string `yaml:"namespace,omitempty"`
}

// TaskView is a unified view of a task regardless of schema version
type TaskView struct {
	ID         string
	Text       string
	Status     string
	AssignedTo string
	DependsOn  []string
	WatchPath  string // legacy compat
	Behavior   AgentBehavior
	IsV11      bool
}
