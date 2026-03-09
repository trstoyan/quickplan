# QuickPlan CLI Architecture

## Overview
QuickPlan CLI is a local-first command-line tool for managing project state, runnable tasks, and optional automation metadata using structured YAML files.

## Core Philosophy
1.  **Local-First:** Work stays on your machine unless you explicitly sync it elsewhere.
2.  **Inspectable State:** Project state is stored in plain files that can be reviewed and versioned.
3.  **Deterministic Execution Contracts:** Runnable tasks must declare how they execute.
4.  **Optional Automation:** Local workers and daemon flows build on the same stored task model.

## Architecture

### 1. State Layer
- **Tool:** `quickplan` (Go CLI)
- **Format:** `tasks.yaml`
- **Responsibility:** Manages tasks, dependencies, agent behaviors, and metadata.

### 2. Execution Layer
- **Communication:** Shared project state (`tasks.yaml` / `project.yaml`) with lock-protected status transitions.
- **Reflexes:** Readiness reconciliation + periodic/fsnotify scans to discover runnable tasks.
- **Execution Contract:** Runnable tasks must define either:
  - `behavior.command` (executed by local/daytona runners)
  - `behavior.plugin` or `assigned_to: plugin:<name>`

### 3. Optional Remote Sync Layer
- **Remote Service:** A compatible remote endpoint may accept pushed or pulled project blueprints.
- **Sync:** `push` and `pull` move project blueprint data without changing local CLI authority over local state.

## Supervisor-Worker Hierarchy
QuickPlan CLI can coordinate a simple two-tier local execution model:

1. **The Worker (The Muscle):**
   - Focuses on atomic task execution.
   - Claims tasks by transitioning them to `IN_PROGRESS`.
   - Executes command/plugin and reports final state (`DONE`/`FAILED`).

2. **The Supervisor (The Brain/Self-Healing):**
   - Monitors project state for `BLOCKED` states.
   - Works with stall detection (`--max-idle`) to avoid silent deadlocks.

## Implementation Details

### Behavior Block
```yaml
behavior:
  role: "Senior Go Architect"
  lifecycle: "Atomic"
  strategy: "TDD"
  command: "go test ./..."
  loop_interval: "30s"
```

### Readiness Checks
Workers use stored task state and local environment checks to determine whether work is runnable.

## Getting Started
1. Install `quickplan` CLI.
2. Create a project: `quickplan create my-app`.
3. Add a runnable task: `quickplan add "Implement Auth" --role "Security Expert" --strategy "Zero Trust" --command "go test ./..."`.
4. Run the daemon: `quickplan daemon`.
5. The daemon will automatically pick up and execute runnable tasks (`TODO`/`PENDING`) that satisfy dependencies and guards.
