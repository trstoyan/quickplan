package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// SendPulse sends a status update to the local web server dashboard
func SendPulse(project, agentID string, taskID interface{}, status, prevStatus string) {
	pulseURL := os.Getenv("QUICKPLAN_WEB_URL")
	if pulseURL == "" {
		pulseURL = "http://localhost:8081"
	}

	pulse := struct {
		Project    string      `json:"project"`
		AgentID    string      `json:"agent_id"`
		TaskID     interface{} `json:"task_id"`
		Status     string      `json:"status"`
		PrevStatus string      `json:"prev_status,omitempty"`
		Timestamp  string      `json:"timestamp"`
	}{
		Project:    project,
		AgentID:    agentID,
		TaskID:     taskID,
		Status:     status,
		PrevStatus: prevStatus,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(pulse)
	if err != nil {
		return
	}

	// Non-blocking pulse send to avoid slowing down the agent
	go func() {
		http.Post(pulseURL+"/api/v1/pulse", "application/json", bytes.NewBuffer(data))
	}()
}

// GenerateSystemPrompt creates the "DNA Handshake" for an agent based on the task and project context.
// This is now a reusable library function.
func GenerateSystemPrompt(t *Task, projectName string) string {
	role := t.Behavior.Role
	if role == "" {
		role = "Senior Software Engineer"
	}
	lifecycle := t.Behavior.LifeCycle
	if lifecycle == "" {
		lifecycle = "Atomic"
	}
	strategy := t.Behavior.Strategy
	if strategy == "" {
		strategy = "Best Practices"
	}

	prompt := fmt.Sprintf("You are the %s for the project '%s'.\n", role, projectName)
	prompt += fmt.Sprintf("Your current lifecycle is %s.\n", lifecycle)
	prompt += fmt.Sprintf("Your task is: %s (ID: %d)\n", t.Text, t.ID)
	prompt += fmt.Sprintf("Strategy: %s\n", strategy)

	if len(t.DependsOn) > 0 {
		prompt += fmt.Sprintf("This task depends on tasks: %v\n", t.DependsOn)
	}

	if t.WatchPath != "" {
		prompt += fmt.Sprintf("Verified environment dependency: %s\n", t.WatchPath)
	}

	prompt += "\nStrict Rule: When finished, explain your changes and output 'STATUS: DONE'."
	return prompt
}
