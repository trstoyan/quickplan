# Archive: Legacy Bash Dependencies

## Removal of qp-loop.sh and qp-guard.sh (v1.2)

In version 1.2, the legacy bash-based worker loops and guard scripts were purged in favor of native Go implementations within the daemon and swarm orchestrator.

### Reasons for Removal
1. **Platform Independence:** Removed dependency on `bash` and `inotify-tools`, allowing for better cross-platform support.
2. **Reliability:** Native Go state transitions are more robust and less prone to race conditions compared to shell script loops.
3. **Performance:** Eliminated the overhead of spawning shell processes and manual pipe management.
4. **Maintainability:** All execution logic is now centralized in the Go codebase.

### Native State Transitions
The daemon now handles task execution using a robust state machine:
- **PENDING/TODO:** Initial runnable state.
- **BLOCKED:** Applied automatically when dependencies/guards fail.
- **IN_PROGRESS:** Set by daemon/swarm before dispatching to a `Runner`.
- **DONE:** Set after successful `Runner` execution.
- **FAILED:** Set if the `Runner` returns an error.
- **RETRYING:** Applied when retry policy allows automatic retry.
- **CANCELLED:** Explicit stop state.

For v1.1 tasks with `retry_policy`, failures can transition `FAILED -> RETRYING -> PENDING` after backoff.

This transition logic is handled atomically by the `ProjectDataManager` to ensure tasks are never executed more than once.
