#!/bin/bash
# name: qp-loop.sh
# description: Reactive loop for sub-agents using inotify.

PROJECT=${1:-"default"}
AGENT_ID=${2:-"agent_$(hostname)"}
BRIDGE_PIPE="/tmp/qp_bridge_${AGENT_ID}"

[ -p "$BRIDGE_PIPE" ] || mkfifo "$BRIDGE_PIPE"

echo "🤖 Agent $AGENT_ID active on Project: $PROJECT"
echo "👂 Listening on $BRIDGE_PIPE and watching for project updates..."

# Get data directory to watch tasks.yaml
DATA_DIR=$(quickplan projects --path | grep "^$PROJECT" | cut -d' ' -f2)
TASKS_FILE="${DATA_DIR}/tasks.yaml"

while true; do
    # 1. Wait for a command from the pipe OR a file system change
    # We use a combined approach: listen to pipe, but also allow external triggers
    if read -t 1 TASK_ID < "$BRIDGE_PIPE"; then
        echo "🚀 Task received from pipe: $TASK_ID"
    else
        # If no pipe input, check for file changes to see if a task was assigned to us
        # This is the 'Reflexive' part
        if inotifywait -q -e modify "$TASKS_FILE" --timeout 1 > /dev/null; then
             echo "🔔 Project updated. Scanning for new assignments..."
             # Logic to find the first PENDING task assigned to this agent
             TASK_ID=$(quickplan list --project "$PROJECT" | grep "PENDING" | grep "$AGENT_ID" | head -n 1 | grep -o "#[0-9]*" | tr -d '#')
        fi
    fi

    if [ ! -z "$TASK_ID" ]; then
        echo "Working on Task $TASK_ID..."
        # Notify dashboard: IN_PROGRESS
        quickplan pulse "$TASK_ID" "IN_PROGRESS" --project "$PROJECT"
        
        sleep 5 
        quickplan complete "$TASK_ID" --project "$PROJECT"
        # Notify dashboard: DONE
        quickplan pulse "$TASK_ID" "DONE" --project "$PROJECT"
        
        echo "✅ Task $TASK_ID completed."
        TASK_ID=""
    fi
done
