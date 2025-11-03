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
# List all available projects (shows current with *)
quickplan projects
```

### Switching Projects

```bash
# Show interactive menu to select project
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

# List tasks in current project
quickplan list

# List tasks in specific project
quickplan list --project work
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

## Task File Format

Tasks are stored in YAML:

```yaml
tasks:
  - id: 1
    text: "Your task description"
    done: false
  - id: 2
    text: "Another task"
    done: false
```

## Tips

- Use quotes around multi-word task descriptions
- Projects are automatically created if using `--project` flag with `add` command
- The interactive menu in `quickplan change` works with arrow keys
- Press Enter to select in the interactive menu
- Your current project persists across shell sessions

## Installation

```bash
# Build from source
make build

# Install system-wide
make install

# Build RPM package
make rpm
```
