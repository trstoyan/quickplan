# Changelog

All notable changes to this project will be documented in this file.

## [0.3.0] - 2026-02-19

### Added
- **Swarm Orchestration**: `quickplan swarm start` command to automate background agent processes.
- **Embedded Scripts**: `qp-loop.sh` and `qp-guard.sh` are now embedded in the binary and extracted at runtime.
- **Interactive Init**: `quickplan init --interactive` wizard for guided project setup.
- **Runner Interface**: Abstracted agent execution logic for future extensibility (tmux/screen support).

## [0.2.0] - 2026-02-19

### Added
- **Multi-Agent Protocol Support**: Implementation of the "Primitive Orchestration" model.
- **Agent Behavior DNA**: New structs in `Task` model for `AssignedTo`, `DependsOn`, `Behavior` (Role, Lifecycle, Strategy), and `WatchPath`.
- **DNA Handshake**: `quickplan agent init` command to generate system prompts for LLM initialization.
- **Registry Sync**: `quickplan sync push/pull` commands for decentralized project blueprint sharing.
- **Project Verification**: `quickplan verify` command to validate project DNA against the registry schema.
- **Reactive Loop Scripts**: Added `qp-loop.sh` (using `inotifywait` and pipes) and `qp-guard.sh` for autonomous execution.
- **Comprehensive Documentation**: Added `ARCHITECTURE.md` and `BLOG_POST.md`.

### Changed
- Refactored `cmd_add.go` to support agent-specific flags (`--assigned-to`, `--role`, `--strategy`, etc.).
- Enhanced `quickplan-web` with registry API endpoints.

## [0.1.0] - 2026-02-15

### Added
- **Project Filtering**: Archived projects are now hidden by default in `quickplan projects` and `quickplan change`.
- **Projects All Flag**: Added `--all` flag to `quickplan projects` to display archived projects.
- **Batch Deletion Support**: `quickplan delete` now supports deleting multiple tasks at once (e.g., `quickplan delete 1 2 3`).
- **Undo Deletion**: New `quickplan undo` command to restore tasks removed in the last deletion.
- **Enhanced Data Storage Reliability**: 
    - Support for `QUICKPLAN_DATADIR` environment variable to override storage location.
    - Automatic fallback to `/tmp/quickplan` if the standard home directory location is read-only or inaccessible.
- **Improved Project Discovery**:
    - Default ignore patterns added for `node_modules` and `build` directories.
    - Automatic creation of `.quickplanignore` with default patterns.
- **Comprehensive Testing**: Added unit tests for deletion logic and ID renumbering.

### Fixed
- Fixed a critical bug where the CLI would crash if the user's home directory was on a read-only filesystem.
- Improved error handling for filesystem permission issues.

### Security
- Data is stored with appropriate 0755/0644 permissions.
