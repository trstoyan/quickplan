# Getting Started with Quick Plan Swarms 🤖

This guide will walk you through setting up your first 4-terminal "Agent Swarm" using the Quick Plan Primitive Orchestration Protocol.

## Prerequisites
- **Arch Linux** (or any Linux with `inotify-tools` and `mkfifo` support)
- **Go** installed
- **Gemini CLI** (or your preferred LLM CLI)

## 1. Install & Initialize
Clone the repo and build the CLI:
```bash
git clone https://github.com/trstoyan/quickplan
cd quick-plan-cli
make build
export PATH=$PATH:$(pwd)/build
```

Initialize your project:
```bash
quickplan init my-swarm --interactive
```

## 2. Configure the Registry
Set your registry endpoint to the new persistent Go backend:
```bash
export QUICKPLAN_REGISTRY_URL="https://registry.quickplan.sh"
```

## 3. Add Runnable Tasks

Workers now require an explicit execution contract per runnable task.

```bash
quickplan add "Run tests" --project my-swarm --command "go test ./..."
quickplan add "Generate docs" --project my-swarm --command "make docs"
```

You can also route work to plugins:

```bash
quickplan add "Security scan" --project my-swarm --plugin secscan
```

## 4. Spawn the Swarm

Use the built-in orchestrator:

```bash
quickplan swarm start --project my-swarm --workers 3 --poll-interval 500ms --max-idle 30s
```

The CLI will:
1. Validate that runnable tasks define `behavior.command` or plugin execution.
2. Start persistent workers that keep claiming runnable tasks.
3. Continue until the project reaches terminal state or stall timeout.

## 5. Advanced: Sandboxed Execution
If you have **Daytona** installed, you can run tasks in isolated environments:

```bash
# In your project.yaml
tasks:
  - name: "Build secure kernel module"
    behavior:
      environment:
        provider: daytona
        image: debian:latest
```

When the worker picks up this task, it will automatically spin up a Daytona workspace, execute the command, and tear it down.

## 6. Run the Global Daemon (Optional)

For continuous background orchestration:

```bash
quickplan daemon
```

The daemon uses the same execution contract and retry/readiness logic as `swarm start`.

## 7. Sync to the Hive
Ready to share your signed blueprint?
```bash
quickplan sync push --project my-swarm
```

Congratulations! You have just orchestrated a zero-overhead, reactive AI swarm using nothing but Go and Linux primitives.
