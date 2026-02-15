# QuickPlan CLI - Project Summary

## Overview

QuickPlan is a lightweight, terminal-based task manager written in Go with project support and vim-inspired selection menus. It provides a fast, keyboard-driven interface for managing tasks across multiple project contexts.

## Key Features

- ✅ **Project-based organization** - Organize tasks into named projects
- ✅ **Context switching** - Easy switching between projects with interactive menu
- ✅ **Vim-inspired UI** - Terminal-native selection interface using Charmbracelet Huh
- ✅ **YAML storage** - Human-readable task storage
- ✅ **RPM packaging** - Ready for RedHat/Fedora/Arch distributions
- ✅ **Single binary** - No dependencies, fast execution

## Architecture

### Language: Go 1.21+

**Why Go?**
- Fast compilation and execution
- Single static binary (8.6MB), no runtime dependencies
- Excellent CLI tool ecosystem
- Great for RPM distribution
- Modern and maintainable code

### Dependencies

- `github.com/spf13/cobra` - CLI framework (commands, flags, help)
- `github.com/charmbracelet/huh` - Interactive forms and selection menus
- `gopkg.in/yaml.v3` - YAML parsing for task storage

### Project Structure

```
quick-plan-cli/
├── main.go              # Entry point, root command, helpers
├── cmd_create.go        # Create project command
├── cmd_change.go        # Change/s switch projects (with menu)
├── cmd_add.go           # Add tasks command
├── cmd_list.go          # List tasks command
├── cmd_projects.go      # List projects command
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── Makefile             # Build automation
├── quickplan.spec       # RPM package specification
├── README.md            # Project documentation
├── USAGE.md             # Usage reference guide
└── .gitignore           # Git ignore rules
```

### Data Storage

All data stored in `~/.local/share/quickplan/`:

```
~/.local/share/quickplan/
├── .current_project     # Currently active project context
├── work/
│   └── tasks.yaml      # Tasks for work project
└── personal/
    └── tasks.yaml      # Tasks for personal project
```

### Commands Implemented

1. **`quickplan create [name] --project`** - Create new projects
2. **`quickplan projects`** - List all available projects
3. **`quickplan change [project]`** - Switch project with interactive menu
4. **`quickplan add [task] --project`** - Add tasks to projects
5. **`quickplan list --project`** - List tasks in projects
6. **`quickplan complete [id]`** - Mark tasks as done
7. **`quickplan delete [id...]`** - Delete one or more tasks
8. **`quickplan undo`** - Undo the last deletion
9. **`quickplan archive [project]`** - Archive projects
10. **`quickplan bdchart`** - Show burndown chart

### Interactive Features

The `quickplan change` command (without arguments) displays an interactive selection menu using Charmbracelet Huh:

- Arrow key navigation
- Enter to select
- Vim-style terminal UI
- Keyboard-driven workflow

### Build System

**Makefile targets:**
- `make build` - Build binary
- `make install` - Install to /usr/local/bin
- `make rpm` - Build RPM package
- `make clean` - Clean artifacts
- `make run` - Run development version

**RPM Package:**
- Spec file: `quickplan.spec`
- Version: 0.1.0
- Installs to `/usr/bin/quickplan`
- Requires: golang >= 1.21 (for building)

## Design Decisions

### 1. Why Not Python?
User requirement: "not using python, we can make it as RPM"

### 2. Why Charmbracelet Huh vs Other Libraries?
- Modern, actively maintained
- Beautiful terminal UI
- Good documentation
- Works great for interactive menus
- No external dependencies in binary

### 3. Why YAML vs JSON/TOML/DB?
- Human-readable and editable
- Go's yaml.v3 is robust
- Simple structure doesn't need complex queries
- Version control friendly

### 4. Why ~/.local/share/quickplan?
- Follows XDG Base Directory specification
- Standard location for user data on Linux
- Easy to backup/migrate
- No sudo required

### 5. Why Per-Project Context?
- Multiple projects need isolation
- Quick switching without losing focus
- State persists across sessions
- Simple implementation

## Future Enhancements (Roadmap)

- [x] Mark tasks as done
- [x] Delete tasks
- [x] Undo/redo support (undo for deletion)
- [ ] Task priorities and due dates
- [ ] Filter and search
- [ ] Export to various formats
- [ ] Import from other task managers
- [ ] Task notes/descriptions
- [ ] Task tags/categories
- [ ] Recurring tasks

## Testing

All core commands tested and working:
- ✅ Project creation
- ✅ Task addition
- ✅ Task listing
- ✅ Project switching
- ✅ Cross-project task operations
- ✅ Data persistence
- ✅ Interactive menu

## Distribution

**Binary size:** 8.6MB (stripped Go binary)

**Supported platforms:** Linux (tested on Arch Linux)

**Installation methods:**
1. Build from source: `make build && make install`
2. RPM package: `make rpm && sudo rpm -i build/rpm/RPMS/*/quickplan-*.rpm`
3. Manual: `go build -o quickplan && sudo mv quickplan /usr/local/bin/`

## License

MIT License - Free and open source

## Development Setup

```bash
# Clone/download source
cd quick-plan-cli

# Install dependencies
go mod tidy

# Build
make build

# Test
./build/quickplan --help
```

## Command Examples

```bash
# Create projects
quickplan create work
quickplan create personal

# List all projects
quickplan projects

# Switch projects (interactive)
quickplan change

# Direct switch
quickplan change work

# Add tasks
quickplan add "Review PR"
quickplan add "Deploy staging" --project work
quickplan add "Buy groceries" --project personal

# List tasks
quickplan list
quickplan list --project work
```

## Conclusion

QuickPlan provides a fast, keyboard-driven task management experience for Linux users who prefer terminal workflows over GUI applications. It's lightweight, dependency-free, and designed with the philosophy of "do one thing well" - managing tasks across projects.
