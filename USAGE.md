
## Integrating with External Agents

QuickPlan's CLI is "Agent-Ready," designed to be driven by external LLMs (like GPT-4, Claude, or local LLaMA).

Note: In bash, ! triggers history expansion even inside double quotes.
Wrap the task in single quotes or escape ! if your text includes it.

### The "Machine Handshake"

External agents can query the current state of a project, make decisions, and execute tasks using the `--json` flag.

**Example: Automated Task Completion Loop**

This script demonstrates how an external agent loop might query QuickPlan, filter for todo tasks using `jq`, and "think" about the next step.

```bash
#!/bin/bash

PROJECT="my-agent-project"

# 1. Query the State (Machine Handshake)
# The agent requests the current task list in JSON format
STATE=$(quickplan list --project "$PROJECT" --json)

# 2. Parse Runnable Tasks
# Use jq to extract tasks that are ready to be worked on
TODO_TASKS=$(echo "$STATE" | jq -r '.tasks[] | select(.status == "TODO" or .status == "PENDING") | .id + ": " + .text')

if [ -z "$TODO_TASKS" ]; then
  echo "No todo tasks. Agent sleeping."
  exit 0
fi

# 3. Agent "Thought" Process (Mocked)
# In a real scenario, $TODO_TASKS would be sent to an LLM API
echo "Agent perceived todo tasks:"
echo "$TODO_TASKS"

# ... LLM processing ...
# ... LLM decides to complete task t-1 ...

# 4. Agent Action
# The agent executes the decision via the CLI
TASK_ID="t-1"
RESPONSE=$(quickplan complete "$TASK_ID" --project "$PROJECT" --json)

# 5. Verify Result
STATUS=$(echo "$RESPONSE" | jq -r '.status')
if [ "$STATUS" == "success" ]; then
  echo "Agent successfully completed task $TASK_ID"
fi
```

## Swarm/Daemon Execution Contract

`quickplan swarm start` and `quickplan daemon` only execute runnable tasks that declare an execution contract:

- `behavior.command`
- `behavior.plugin` or `assigned_to: plugin:<name>`

Create runnable tasks directly from CLI:

```bash
quickplan add "Run tests" --project "$PROJECT" --command "go test ./..."
quickplan add "Security scan" --project "$PROJECT" --plugin secscan
```

Run the swarm until completion (or stall timeout):

```bash
quickplan swarm start --project "$PROJECT" --workers 3 --poll-interval 500ms --max-idle 30s
```
