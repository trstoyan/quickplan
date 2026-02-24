
## Integrating with External Agents

QuickPlan's CLI is "Agent-Ready," designed to be driven by external LLMs (like GPT-4, Claude, or local LLaMA).

### The "Machine Handshake"

External agents can query the current state of a project, make decisions, and execute tasks using the `--json` flag.

**Example: Automated Task Completion Loop**

This script demonstrates how an external agent loop might query QuickPlan, filter for pending tasks using `jq`, and "think" about the next step.

```bash
#!/bin/bash

PROJECT="my-agent-project"

# 1. Query the State (Machine Handshake)
# The agent requests the current task list in JSON format
STATE=$(quickplan list --project "$PROJECT" --json)

# 2. Parse Pending Tasks
# Use jq to extract tasks that are not done
PENDING_TASKS=$(echo "$STATE" | jq -r '.tasks[] | select(.status == "PENDING") | .id + ": " + .text')

if [ -z "$PENDING_TASKS" ]; then
  echo "No pending tasks. Agent sleeping."
  exit 0
fi

# 3. Agent "Thought" Process (Mocked)
# In a real scenario, $PENDING_TASKS would be sent to an LLM API
echo "Agent perceived pending tasks:"
echo "$PENDING_TASKS"

# ... LLM processing ...
# ... LLM decides to complete task t-1 ...

# 4. Agent Action
# The agent executes the decision via the CLI
TASK_ID="t-1"
RESPONSE=$(quickplan complete "$TASK_ID" --project "$PROJECT" --json)

# 5. Verify Result
SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
if [ "$SUCCESS" == "true" ]; then
  echo "Agent successfully completed task $TASK_ID"
fi
```
