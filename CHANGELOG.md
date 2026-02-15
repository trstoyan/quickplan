# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2026-02-15

### Added
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
