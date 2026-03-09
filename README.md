# QuickPlan CLI

[![Support the Project](https://img.shields.io/badge/Support-Buy%20Me%20A%20Coffee-yellow.svg)](https://buymeacoffee.com/trstoyan)

A fast, lightweight CLI task manager for Linux with project support and vim-inspired selection menus.

## Features

- 📁 **Project-based organization** - Organize tasks into named projects
- 🎯 **Context switching** - Easily switch between projects
- 🖥️ **Vim-inspired menus** - Terminal-native selection interface
- ✓ **Task completion** - Mark tasks as done with timestamps
- 🗑️ **Delete tasks** - Remove tasks with confirmation
- 📊 **Smart task listing** - Completed tasks shown separately at bottom (latest 5 by default)
- 🌐 **Multi-project view** - List tasks from all projects at once
- 📦 **Archive projects** - Archive completed or inactive projects
- ⏰ **Timestamps** - Track creation and modification times
- 🔄 **Version tracking** - Automatic version migration for project data
- 🚫 **Ignore patterns** - Customize which directories to ignore (`.quickplanignore`)
- 🔗 **Sync source config** - Optional metadata for future remote sync workflows
- 📈 **Burndown charts** - Visualize task completion progress over time
- ⚡ **Fast & lightweight** - Written in Go, single binary
- 📦 **RPM packaging** - Easy installation on RPM-based systems

## Installation

### Build from Source

```bash
# Clone or download the source
git clone https://github.com/trstoyan/quickplan.git
cd quickplan

# Build the binary
make build

# Install to /usr/local/bin
make install
```

### Build RPM Package

```bash
# Build RPM
make rpm

# Install RPM
sudo rpm -i build/rpm/RPMS/*/quickplan-0.1.0-1.*.rpm
```

### Manual Installation

```bash
go build -o quickplan
sudo mv quickplan /usr/local/bin/
```

## Usage

### Create a New Project

```bash
# Create a project with a specific name
quickplan create myproject

# Create using the --project flag
quickplan create --project work

# Create a default project
quickplan create
```

### List Projects

```bash
# List only active projects and current one
quickplan projects

# List all projects including archived ones
quickplan projects --all
```

### Switch Between Projects

```bash
# Show interactive menu to select project
quickplan change

# Switch directly to a project
quickplan change myproject
```

### Add Tasks

```bash
# Add to current project
quickplan add "Complete the feature documentation"

# Add to a specific project
quickplan add "Review pull request" --project work

# Add an executable task for swarm/daemon workers
quickplan add "Run unit tests" --command "go test ./..."

# Add a plugin-driven task
quickplan add "Run security checks" --plugin secscan

# Multiple tasks
quickplan add "Task one"
quickplan add "Task two"
```

Note: In bash, ! triggers history expansion even inside double quotes.
Wrap the task in single quotes or escape ! if your text includes it.

### Complete Tasks

```bash
# Mark a task as complete (interactive menu)
quickplan complete

# Complete specific task by ID
quickplan complete 1

# Complete task in specific project
quickplan complete 2 --project work

# Add a note when completing
quickplan complete 1 --note "Reviewed and approved"
```

### Delete Tasks

```bash
# Delete a task by ID (with confirmation)
quickplan delete 1

# Delete multiple tasks at once
quickplan delete 1 2 3

# Delete task in specific project
quickplan delete 3 --project work

# Force delete without confirmation
quickplan delete 2 --force
```

### Undo Deletion

```bash
# Restore tasks removed in the last delete command
quickplan undo
```

### List Tasks

```bash
# List incomplete tasks in current project
quickplan list

# List all tasks including completed (shows latest 5 completed at bottom)
quickplan list --all

# List tasks in a specific project
quickplan list --project work --all

# List tasks from all projects
quickplan list --all-projects

# List all tasks from all projects (including all completed)
quickplan list --all-projects --all
```

### Archive Projects

```bash
# Archive the current project
quickplan archive

# Archive a specific project
quickplan archive old-project

# Unarchive (toggle off)
quickplan archive old-project
```

### Burndown Charts

```bash
# Display a text-based burndown chart for the current project
quickplan bdchart
```

Visualize task completion progress over time with a simple ASCII chart showing incomplete tasks per day.

### Swarm and Daemon Execution Contract

`quickplan swarm start` and `quickplan daemon` now execute real task work only when each runnable task has an execution contract:

- `behavior.command` (shell command)
- `behavior.plugin` or `assigned_to: plugin:<name>`

If a runnable task has neither, swarm startup fails fast with a validation error.

```bash
# Example runnable task
quickplan add "Build binary" --command "go build ./..."

# Start workers until terminal state (DONE/FAILED/CANCELLED)
quickplan swarm start --workers 3 --poll-interval 500ms --max-idle 30s

# Background engine with the same execution contract rules
quickplan daemon
```

Local command execution uses `sh -lc`, so shell operators such as `&&`, `|`, redirects, and quoting are supported.

### Ignore Patterns

QuickPlan automatically ignores certain directories like `.git` when listing projects. You can customize this behavior:

```bash
# Create a .quickplanignore file in ~/.local/share/quickplan/
# Add patterns (one per line, supports glob matching):
#
# .git
# temp
# backup-*
# test_*
```

Default ignored patterns:
- `.git` - Git repository directories
- `.*` - All hidden directories (starting with dot)
- `node_modules` - Node.js dependencies
- `build` - Build artifacts
- `.current_project` - Current project marker used by the CLI

### Project Management Summary

```bash
# See all projects
quickplan projects

# Switch to a project
quickplan change work

# Add tasks to current or specific project
quickplan add "Task description"
quickplan add "Task" --project work --command "echo done"

# View tasks (incomplete only, latest 5 completed shown at bottom)
quickplan list

# View all tasks including completed
quickplan list --all

# View tasks from all projects
quickplan list --all-projects

# Complete tasks
quickplan complete 1

# Archive finished projects
quickplan archive old-project
```

The current project is stored in `~/.local/share/quickplan/.current_project`

## Project Structure

```
~/.local/share/quickplan/
├── .current_project          # Currently active project
├── .quickplanignore          # Custom ignore patterns (optional)
├── default/
│   ├── tasks.yaml           # Tasks for default project
│   └── project.yml          # Project configuration
├── work/
│   ├── tasks.yaml           # Tasks for work project
│   └── project.yml          # Project configuration
└── myproject/
    ├── tasks.yaml           # Tasks for myproject
    └── project.yml          # Project configuration
```

## File Formats

### tasks.yaml

Tasks are stored in YAML format with timestamps and version tracking:

```yaml
quickplan-cli-version: "0.1.0"
tasks:
  - id: 1
    text: "Complete the documentation"
    done: true
    created: 2025-11-03T13:07:12Z
    completed: 2025-11-03T14:30:00Z
  - id: 2
    text: "Review code changes and run tests"
    done: false
    status: "TODO"
    behavior:
      role: "Reviewer"
      command: "go test ./..."
    created: 2025-11-03T13:08:00Z
created: 2025-11-03T13:07:12Z
modified: 2025-11-03T14:30:00Z
archived: false
```

### project.yml

Project configuration for future sync capabilities:

```yaml
name: "myproject"
description: "My awesome project"
sync_source:
  type: "local"  # Options: local, git, server
  url: ""        # For git: repo URL, for server: remote service URL
  branch: ""     # For git sources
created: 2025-11-03T13:07:12Z
modified: 2025-11-03T14:30:00Z
```

**Note:** The `sync_source` configuration is optional. It allows projects to record where they may later be synced from or published to, without changing the local-first CLI workflow.

## Development

### Requirements

- Go 1.21 or later
- Make (for building RPM)

### Build

```bash
# Clone/download source
git clone https://github.com/trstoyan/quickplan.git
cd quickplan

# Build binary
make build

# Clean artifacts
make clean

# Run application
make run

# Run tests
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

### Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/huh` - Interactive forms and menus
- `gopkg.in/yaml.v3` - YAML parsing

## Project Direction

QuickPlan CLI is maintained as a standalone, local-first tool.

Public priorities for this repository are:
- stable local task and project workflows
- clear file formats and migration behavior
- reliable local execution contracts for runnable tasks
- optional remote sync that does not break standalone usage
- small, reviewable changes with good test coverage

## Commercial vs. Open Source

QuickPlan follows an **Open Core** model:

- **QuickPlan CLI (MIT)**: This repository contains the open-source command-line tool for local-first project and task workflows.
- **Hosted / Managed Offerings**: Separately operated services may provide remote sync, collaboration, or managed automation. Those services are not included in this repository.

## Optional Remote Integration

When connecting to a compatible remote service, the CLI uses these environment variables:
- `QUICKPLAN_REGISTRY_URL` for the remote base URL used by `quickplan sync` and remote health checks
- `QUICKPLAN_REMOTE_TOKEN` for bearer authentication
- `QUICKPLAN_API_KEY` for API-key authentication when supported by the remote service

`QUICKPLAN_REMOTE_TOKEN` is the preferred public bearer-token variable. `QUICKPLAN_WEB_TOKEN` remains supported as a legacy fallback for older local setups.

## License

MIT License - See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
