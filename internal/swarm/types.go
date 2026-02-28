package swarm

// EnvironmentConfig defines the execution environment for an agent.
type EnvironmentConfig struct {
	Provider string `yaml:"provider,omitempty"` // "local", "daytona"
	Image    string `yaml:"image,omitempty"`    // e.g., "golang:1.22"
}

// AgentBehavior defines the "personality" and "loop rules" for an AI agent.
type AgentBehavior struct {
	Role         string            `yaml:"role,omitempty"`          // e.g., "Senior Go Architect"
	LifeCycle    string            `yaml:"lifecycle,omitempty"`     // e.g., "Atomic" (one-shot) or "Infinite" (loop)
	LoopInterval string            `yaml:"loop_interval,omitempty"` // e.g., "30s"
	Strategy     string            `yaml:"strategy,omitempty"`      // e.g., "TDD" or "Fast Prototype"
	Command      string            `yaml:"command,omitempty"`       // shell command for task execution
	Plugin       string            `yaml:"plugin,omitempty"`        // plugin executable name
	Environment  EnvironmentConfig `yaml:"environment,omitempty"`
}

// TaskView is a unified view of a task regardless of schema version.
type TaskView struct {
	ID            string
	Text          string
	Status        string
	AssignedTo    string
	DependsOn     []string
	WatchPath     string // legacy compat
	WatchPaths    []string
	RequiresFiles []string
	Behavior      AgentBehavior
	IsV11         bool
}
