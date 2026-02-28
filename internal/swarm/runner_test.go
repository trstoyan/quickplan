package swarm

import (
	"strings"
	"testing"
)

func TestLocalRunnerExecute_SupportsShellOperators(t *testing.T) {
	t.Setenv("QUICKPLAN_DISABLE_LOCAL_SANDBOX", "1")

	runner := &LocalRunner{Project: "p", AgentID: "a"}
	task := &TaskView{ID: "t-1"}

	out, err := runner.Execute("echo one && echo two", task)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !strings.Contains(out, "one") || !strings.Contains(out, "two") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestLocalRunnerExecute_EmptyCommandFails(t *testing.T) {
	runner := &LocalRunner{Project: "p", AgentID: "a"}
	task := &TaskView{ID: "t-1"}

	if _, err := runner.Execute("", task); err == nil {
		t.Fatal("expected empty command error")
	}
}
