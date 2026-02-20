package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor swarm pulses in real-time",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFilter, _ := cmd.Flags().GetString("project")
		baseURL, _ := cmd.Flags().GetString("url")
		asJSON, _ := cmd.Flags().GetBool("json")

		if baseURL == "" {
			baseURL = "http://localhost:8080"
		}

		streamURL := fmt.Sprintf("%s/api/v1/pulse/stream", baseURL)
		
		fmt.Printf("📡 Connecting to monitor stream at %s...\n", streamURL)
		if projectFilter != "" {
			fmt.Printf("🎯 Filtering for project: %s\n", projectFilter)
		}

		resp, err := http.Get(streamURL)
		if err != nil {
			return fmt.Errorf("failed to connect to stream: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server returned error: %s", resp.Status)
		}

		// Handle interrupt for clean exit
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		
		go func() {
			<-sigChan
			fmt.Println("\n👋 Disconnecting from monitor...")
			resp.Body.Close()
			os.Exit(0)
		}()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("stream connection closed: %w", err)
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}

			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)

			var pulse SwarmPulse
			if err := json.Unmarshal([]byte(data), &pulse); err != nil {
				// Might be a non-pulse SSE message or keepalive
				continue
			}

			// Filter by project if requested
			if projectFilter != "" && pulse.Project != projectFilter {
				continue
			}

			if asJSON {
				fmt.Println(data)
			} else {
				// Format: <ts> <agent> <task_id> <status>
				ts := pulse.Timestamp
				if len(ts) > 19 {
					ts = ts[11:19] // Just time part HH:MM:SS
				}
				fmt.Printf("[%s] %-15s | Task %-5s | %s\n", ts, pulse.AgentID, pulse.TaskID, pulse.Status)
			}
		}
	},
}

func init() {
	monitorCmd.Flags().StringP("project", "p", "", "Filter by project name")
	monitorCmd.Flags().String("url", "http://localhost:8080", "Registry base URL")
	monitorCmd.Flags().Bool("json", false, "Output as raw JSON lines")
}

type SwarmPulse struct {
	Project   string `json:"project"`
	AgentID   string `json:"agent_id"`
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}
