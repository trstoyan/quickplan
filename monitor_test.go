package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestSSEParsing(t *testing.T) {
	input := `data: {"project":"p1", "agent_id":"a1", "task_id":"t1", "status":"DONE", "timestamp":"2026-02-20T12:00:00Z"}

:keepalive

data: {"project":"p1", "agent_id":"a2", "task_id":"t2", "status":"FAILED"}
`
	
	reader := bufio.NewReader(strings.NewReader(input))
	
	var count int
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		
		count++
	}
	
	if count != 2 {
		t.Errorf("expected 2 pulses parsed, got %d", count)
	}
}
