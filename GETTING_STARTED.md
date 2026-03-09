# Getting Started with QuickPlan CLI

This guide walks through a local-first setup for QuickPlan CLI. It focuses on standalone usage first, then shows where optional remote sync can fit in later.

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

## 2. Add Runnable Tasks

Workers now require an explicit execution contract per runnable task.

```bash
quickplan add "Run tests" --project my-swarm --command "go test ./..."
quickplan add "Generate docs" --project my-swarm --command "make docs"
```

You can also route work to plugins:

```bash
quickplan add "Security scan" --project my-swarm --plugin secscan
```

## 3. Run Workers

Use the built-in orchestrator:

```bash
quickplan swarm start --project my-swarm --workers 3 --poll-interval 500ms --max-idle 30s
```

The CLI will:
1. Validate that runnable tasks define `behavior.command` or plugin execution.
2. Start persistent workers that keep claiming runnable tasks.
3. Continue until the project reaches terminal state or stall timeout.

## 4. Advanced: Sandboxed Execution
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

## 5. Run the Global Daemon (Optional)

For continuous background orchestration:

```bash
quickplan daemon
```

The daemon uses the same execution contract and retry/readiness logic as `swarm start`.

## 6. Optional Remote Sync

QuickPlan CLI can also push and pull project blueprints when you point it at a compatible remote service:

```bash
export QUICKPLAN_REGISTRY_URL="https://your-service.example"
quickplan sync push --project my-swarm
```

If you do not configure a remote service, QuickPlan CLI still works as a fully local-first tool.
