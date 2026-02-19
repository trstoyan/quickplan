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
- **Communication:** Named Pipes (`mkfifo`) for real-time task dispatching.
- **Reflexes:** `inotify` triggers for reactive execution on file changes.
- **DNA Handshake:** `quickplan agent init` generates system prompts that define agent roles and constraints.

### 3. Network Layer (The Hive)
- **Registry:** `quickplan.sh` (Central/Decentralized server)
- **Sync:** `push` and `pull` project blueprints and agent behaviors.

## Supervisor-Worker Hierarchy
Quick Plan utilizes a two-tier management structure:

1. **The Worker (The Muscle):**
   - Focuses on atomic task execution.
   - Reports status (DONE/BLOCKED) to the Blackboard.
   - Communicates via pipes and inotify.

2. **The Supervisor (The Brain/Self-Healing):**
   - Monitors the Blackboard for `BLOCKED` states.
   - Analyzes blocker reasons using a "Correction Agent".
   - Injects new sub-tasks into the plan to resolve dependencies or errors.
   - Ensures the swarm never deadlocks.

## Implementation Details

### Agent DNA (Behavior Block)
```yaml
behavior:
  role: "Senior Go Architect"
  lifecycle: "Atomic"
  strategy: "TDD"
  loop_interval: "30s"
```

### Active Awareness
Agents use `qp-guard.sh` to verify that both logical (task status) and physical (file existence) dependencies are met before proceeding.

## Getting Started
1. Install `quickplan` CLI.
2. Create a project: `quickplan create my-app`.
3. Add a task with agent behavior: `quickplan add "Implement Auth" --role "Security Expert" --strategy "Zero Trust"`.
4. Initialize an agent loop: `./scripts/qp-loop.sh my-app agent-01`.
5. Dispatch a task: `echo "1" > /tmp/qp_bridge_agent-01`.
