# QuickPlan CLI - Quick Reference

## Basic Commands

### Creating Projects

```bash
# Create a project named "work"
quickplan create work

# Using --project flag
quickplan create --project personal

# Create default project
quickplan create
```

### Listing Projects

```bash
# List only active projects (shows current with *)
quickplan projects

# List all projects including archived ones
quickplan projects --all
```

### Switching Projects

```bash
# Show interactive menu to select from active projects
quickplan change

# Switch directly
quickplan change work
```

### Managing Tasks

```bash
# Add task to current project
quickplan add "Complete the documentation"

# Add task to specific project
quickplan add "Review PR" --project work

# Mark a task as complete
quickplan complete 1

# List tasks in current project (incomplete + latest 5 completed)
quickplan list

# List all tasks including all completed
quickplan list --all

# List tasks in specific project
quickplan list --project work

# Delete a task
quickplan delete 1

# Delete multiple tasks
quickplan delete 1 2 3

# Undo the last deletion
quickplan undo

# Archive/unarchive current project
quickplan archive

# Archive/unarchive specific project
quickplan archive old-project

# Show burndown chart for current project
quickplan bdchart

# List tasks from all projects
quickplan list --all-projects

# List all tasks from all projects
quickplan list --all-projects --all
```

## Workflow Examples

### Starting a New Project

```bash
# 1. Create your project
quickplan create myproject

# 2. Add some tasks
quickplan add "Set up database"
quickplan add "Create API endpoints"
quickplan add "Write tests"

# 3. List your tasks
quickplan list
```

### Working with Multiple Projects

```bash
# Create two projects
quickplan create work
quickplan create personal

# List all projects
quickplan projects

# Switch between them
quickplan change work
quickplan add "Finish report"

quickplan change personal
quickplan add "Call dentist"

# View tasks in each
quickplan list --project work
quickplan list --project personal
```

### Adding Tasks to Different Projects

```bash
# Without specifying project (uses current)
quickplan add "Default task"

# To a specific project regardless of current
quickplan add "Work task" --project work
quickplan add "Personal task" --project personal
```

## Data Storage

All data is stored in `~/.local/share/quickplan/`:

```
~/.local/share/quickplan/
├── .current_project       # Your active project
├── work/
│   └── tasks.yaml        # Work project tasks
└── personal/
    └── tasks.yaml        # Personal project tasks
```

## Swarm Orchestration (v0.3.0)

Automate your multi-agent workflow with the Swarm Orchestrator.

### Initialization

```bash
# Interactive Wizard
quickplan init --interactive
```

This will guide you through:
1. Creating a project.
2. Seeding initial tasks with agent roles.

### Starting a Swarm

```bash
# Start 3 worker agents for the current project
quickplan swarm start --workers 3

# Start 5 workers for a specific project
quickplan swarm start --workers 5 --project my-app
```

The orchestrator will:
1. Extract the necessary execution scripts (`qp-loop.sh`).
2. Spawn background processes for each worker.
3. Agents will automatically pick up tasks assigned to them or broadcast to the swarm.

## Installation

```bash
# Build from source
make build

# Install system-wide
make install

# Build RPM package
make rpm
```
