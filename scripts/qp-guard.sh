#!/bin/bash
# name: qp-guard.sh
# description: Checks for dependencies before allowing an agent to proceed.

TASK_ID=$1
DEPENDENCY_FILE=$2

if [ -z "$TASK_ID" ]; then
    echo "Usage: $0 <task_id> [dependency_file]"
    exit 1
fi

echo "🛡️ Guard active for Task: $TASK_ID"

# 1. Check Quick Plan for 'DONE' status
# Note: We assume quickplan is in the PATH
STATUS=$(quickplan list --project default | grep "^\[X\] #$TASK_ID" > /dev/null && echo "DONE" || echo "PENDING")

# A better way is to use a specific command if we implement it, 
# but for now we can use grep on the list output.
# Let's assume a future 'quickplan task show' command.

if [ "$STATUS" != "DONE" ]; then
    echo "⚠️ Dependency Task #$TASK_ID is not marked DONE in Quick Plan."
    exit 1
fi

# 2. Physical Check: Does the file actually exist?
if [ ! -z "$DEPENDENCY_FILE" ]; then
    if [ ! -f "$DEPENDENCY_FILE" ] && [ ! -d "$DEPENDENCY_FILE" ]; then
        echo "❌ Status is DONE, but $DEPENDENCY_FILE is missing! Inconsistency detected."
        quickplan note add "Auto-check failed: Path $DEPENDENCY_FILE not found."
        exit 1
    fi
fi

echo "✅ All clear. Proceeding..."
exit 0
