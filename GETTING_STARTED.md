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

## 3. Spawn the Swarm (The Easy Way)
Instead of manual terminal windows, use the built-in orchestrator:

```bash
quickplan swarm start --workers 3
```

The CLI will:
1. Extract the `qp-loop.sh` worker logic.
2. Initialize the background agent pool.
3. Start the **Supervisor** to monitor for blocked tasks.

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

## 5. Sync to the Hive
Ready to share your signed blueprint?
```bash
quickplan sync push --project my-swarm
```

Congratulations! You have just orchestrated a zero-overhead, reactive AI swarm using nothing but Go and Linux primitives.
