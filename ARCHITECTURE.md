# Quick Plan: Multi-Agent Orchestration Protocol

## Overview
Quick Plan is a decentralized, local-first orchestration protocol that treats project management as a shared "Blackboard". It enables humans and AI agents to collaborate seamlessly using a structured YAML state.

## Core Philosophy
1.  **Local-First & Private:** Work stays on your machine until you choose to sync it.
2.  **Primitive Orchestration:** Leverages Linux native features (Pipes, inotify, signals) for low overhead and high reliability.
3.  **Vendor-Agnostic:** Compatible with any LLM CLI (Gemini, Claude, Codex, etc.).
4.  **Living Plan:** Agents don't just follow the plan; they verify the environment and update the plan dynamically.

## Architecture

### 1. State Layer (The Blackboard)
- **Tool:** `quickplan` (Go CLI)
- **Format:** `tasks.yaml`
- **Responsibility:** Manages tasks, dependencies, agent behaviors, and metadata.

### 2. Execution Layer (The Nerves)
- **Communication:** Shared project state (`tasks.yaml` / `project.yaml`) with lock-protected status transitions.
- **Reflexes:** Readiness reconciliation + periodic/fsnotify scans to discover runnable tasks.
- **DNA Handshake:** `quickplan agent init` generates system prompts that define agent roles and constraints.
- **Execution Contract:** Runnable tasks must define either:
  - `behavior.command` (executed by local/daytona runners)
  - `behavior.plugin` or `assigned_to: plugin:<name>`

### 3. Network Layer (The Hive)
- **Registry:** `quickplan.sh` (Central/Decentralized server)
- **Sync:** `push` and `pull` project blueprints and agent behaviors.

## Supervisor-Worker Hierarchy
Quick Plan utilizes a two-tier management structure:

1. **The Worker (The Muscle):**
   - Focuses on atomic task execution.
   - Claims tasks by transitioning them to `IN_PROGRESS`.
   - Executes command/plugin and reports final state (`DONE`/`FAILED`).

2. **The Supervisor (The Brain/Self-Healing):**
   - Monitors the Blackboard for `BLOCKED` states.
   - Analyzes blocker reasons using a "Correction Agent".
   - Injects new sub-tasks into the plan to resolve dependencies or errors.
   - Works with stall detection (`--max-idle`) to avoid silent deadlocks.

## Implementation Details

### Agent DNA (Behavior Block)
```yaml
behavior:
  role: "Senior Go Architect"
  lifecycle: "Atomic"
  strategy: "TDD"
  command: "go test ./..."
  loop_interval: "30s"
```

### Active Awareness
Agents use native state verification to ensure that both logical (task status) and physical (file existence) dependencies are met before proceeding.

## Getting Started
1. Install `quickplan` CLI.
2. Create a project: `quickplan create my-app`.
3. Add a runnable task: `quickplan add "Implement Auth" --role "Security Expert" --strategy "Zero Trust" --command "go test ./..."`.
4. Run the daemon: `quickplan daemon`.
5. The daemon will automatically pick up and execute runnable tasks (`TODO`/`PENDING`) that satisfy dependencies and guards.
