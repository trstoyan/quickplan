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
go build -o build/quickplan .
export PATH=$PATH:$(pwd)/build
```

Initialize your first project:
```bash
quickplan create my-swarm
```

## 2. Define the Plan
Add a few tasks with agent behaviors. We'll set up a dependency where Task 2 waits for Task 1.
```bash
quickplan add "Write the API schema" --role "Backend Architect" --strategy "REST"
quickplan add "Generate Frontend types" --role "Frontend Lead" --depends-on 1 --watch-path "schema.graphql"
```

## 3. Spawn the Swarm
Open **three new terminal windows** (or use tmux). We will assign two worker agents and one observer.

**Terminal 2 (Agent Alpha):**
```bash
./scripts/qp-loop.sh my-swarm alpha
```

**Terminal 3 (Agent Beta):**
```bash
./scripts/qp-loop.sh my-swarm beta
```

**Terminal 4 (The Master Observer):**
```bash
watch -n 1 quickplan list --project my-swarm
```

## 4. Dispatch and Execute
In your main terminal, send the first task to Alpha:
```bash
echo "1" > /tmp/qp_bridge_alpha
```

**What happens next?**
1. **Alpha** wakes up, sees the task, and simulates the work (saving `schema.graphql`).
2. **Alpha** updates the task to `DONE`.
3. **Beta**'s `inotify` reflex triggers. It scans the plan, sees that Task 1 is `DONE` and `schema.graphql` exists.
4. **Beta** begins working on Task 2 automatically.

## 5. Sync to the Hive
Ready to share your blueprint?
```bash
quickplan sync push --project my-swarm
```

Congratulations! You have just orchestrated a zero-overhead, reactive AI swarm using nothing but Go and Linux primitives.
