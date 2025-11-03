# QuickPlan CLI

A fast, lightweight CLI task manager for Linux with project support and vim-inspired selection menus.

## Features

- 📁 **Project-based organization** - Organize tasks into named projects
- 🎯 **Context switching** - Easily switch between projects
- 🖥️ **Vim-inspired menus** - Terminal-native selection interface
- ✓ **Task completion** - Mark tasks as done with timestamps
- 📦 **Archive projects** - Archive completed or inactive projects
- ⏰ **Timestamps** - Track creation and modification times
- ⚡ **Fast & lightweight** - Written in Go, single binary
- 📦 **RPM packaging** - Easy installation on RPM-based systems

## Installation

### Build from Source

```bash
# Clone or download the source
cd quick-plan-cli

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
# List all available projects and current one
quickplan projects
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

# Multiple tasks
quickplan add "Task one"
quickplan add "Task two"
```

### Complete Tasks

```bash
# Mark a task as complete (interactive menu)
quickplan complete

# Complete specific task by ID
quickplan complete 1

# Complete task in specific project
quickplan complete 2 --project work
```

### List Tasks

```bash
# List incomplete tasks in current project
quickplan list

# List all tasks including completed
quickplan list --all

# List tasks in a specific project
quickplan list --project work --all
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

### Project Management Summary

```bash
# See all projects
quickplan projects

# Switch to a project
quickplan change work

# Add tasks to current or specific project
quickplan add "Task description"
quickplan add "Task" --project work

# View tasks (incomplete only)
quickplan list

# View all tasks including completed
quickplan list --all

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
├── default/
│   └── tasks.yaml           # Tasks for default project
├── work/
│   └── tasks.yaml           # Tasks for work project
└── myproject/
    └── tasks.yaml           # Tasks for myproject
```

## Tasks File Format

Tasks are stored in YAML format with timestamps:

```yaml
tasks:
  - id: 1
    text: "Complete the documentation"
    done: true
    created: 2025-11-03T13:07:12Z
    completed: 2025-11-03T14:30:00Z
  - id: 2
    text: "Review code changes"
    done: false
    created: 2025-11-03T13:08:00Z
created: 2025-11-03T13:07:12Z
modified: 2025-11-03T14:30:00Z
archived: false
```

## Development

### Requirements

- Go 1.21 or later
- Make (for building RPM)

### Build

```bash
# Build binary
make build

# Clean artifacts
make clean

# Run application
make run
```

### Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/huh` - Interactive forms and menus
- `gopkg.in/yaml.v3` - YAML parsing

## Roadmap

- [x] List tasks with completion status
- [x] Mark tasks as done with timestamps
- [x] Archive projects
- [x] Project and task timestamps
- [ ] Delete tasks and projects
- [ ] Filter and search tasks
- [ ] Export tasks to various formats
- [ ] Import from other task managers
- [ ] Task priorities and due dates
- [ ] Task notes and descriptions

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
