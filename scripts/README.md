# Quick Plan Automation Scripts

These scripts provide the "Reflexes" and "Guards" for the Multi-Agent Orchestration Protocol.

## qp-loop.sh
The primary execution loop for an autonomous agent.

- **Input:** Project name (e.g., `default`), Agent ID (e.g., `agent-01`).
- **Output:** Execution status, task completion updates.
- **Mechanism:** 
  - Uses `mkfifo` to create a dedicated communication bridge at `/tmp/qp_bridge_<agent_id>`.
  - Uses `inotifywait` to reactively scan the project's `tasks.yaml` when a modification is detected.
  - Blocks on `read` from the pipe to ensure zero CPU overhead when idle.
- **Dependencies:** `inotify-tools`, `quickplan` CLI.

## qp-guard.sh
A logical and physical validator for task dependencies.

- **Input:** Task ID (integer), Dependency Path (optional file/directory).
- **Output:** Exit code 0 if all clear, exit code 1 if dependencies are missing.
- **Mechanism:**
  - Checks if the specified Task ID is marked as `DONE` in the project state.
  - If a Dependency Path is provided, it verifies the actual existence of that path in the filesystem.
- **Dependencies:** `quickplan` CLI.

## Usage
To start an agent swarm:
1. Open a terminal and run: `./qp-loop.sh my-project agent-alpha`
2. Open another terminal and assign a task: `echo "1" > /tmp/qp_bridge_agent-alpha`
