package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/trstoyan/quickplan/internal/swarm"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the global background engine",
	Long:  `The daemon runs as a background process, monitoring all active projects and executing tasks autonomously.`,
	RunE:  runDaemon,
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}

func runDaemon(cmd *cobra.Command, args []string) error {
	// 1. Initialize Logger
	dataDir, err := getDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	logPath := filepath.Join(dataDir, "events.jsonl")
	logger, err := swarm.NewEventLogger(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		return err
	}
	defer logger.Close()
	logger.OutputJSON = globalJSON

	logger.Log("INFO", "Daemon", "Starting QuickPlan Daemon", map[string]interface{}{
		"version": version,
		"pid":     os.Getpid(),
	})

	// 2. Initialize Managers
	versionManager := NewVersionManager(version)
	projectManager := NewProjectDataManager(dataDir, versionManager)

	// 3. Setup FSNotify
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log("ERROR", "Daemon", "Failed to create watcher", map[string]interface{}{"error": err.Error()})
		return err
	}
	defer watcher.Close()

	// Track which directories we are watching
	watchedDirs := make(map[string]bool)
	var watchMu sync.Mutex

	// Helper to add project directories to watcher
	addProjectWatches := func() {
		projects, err := projectManager.ListProjects(false)
		if err != nil {
			return
		}

		watchMu.Lock()
		defer watchMu.Unlock()
		for _, p := range projects {
			pDir := filepath.Join(dataDir, p)
			if !watchedDirs[pDir] {
				if err := watcher.Add(pDir); err == nil {
					watchedDirs[pDir] = true
					logger.Log("INFO", "Daemon", fmt.Sprintf("Watching project: %s", p), nil)
				}
			}
		}
		// Also watch the root dataDir for new project creation
		if !watchedDirs[dataDir] {
			if err := watcher.Add(dataDir); err == nil {
				watchedDirs[dataDir] = true
			}
		}
	}

	addProjectWatches()

	// 4. Task Execution Engine
	activeAgents := make(map[string]int)
	var agentMu sync.Mutex
	const maxAgentsPerProject = 2

	processProject := func(project string) {
		// Check if we have capacity for this project
		agentMu.Lock()
		count := activeAgents[project]
		agentMu.Unlock()

		if count >= maxAgentsPerProject {
			return
		}

		// Check for tasks with Status: "TODO"
		views, _, err := projectManager.GetTaskViews(project)
		if err != nil {
			return
		}

		var targetTask *TaskView
		for _, v := range views {
			if v.Status == "TODO" && v.AssignedTo == "" {
				targetTask = &v
				break
			}
		}

		if targetTask != nil {
			agentID := fmt.Sprintf("daemon-worker-%d", time.Now().UnixNano()%10000)

			// Update state to IN_PROGRESS and save to prevent hot-loop/double-grabbing
			if err := projectManager.UpdateTaskStatus(project, targetTask.ID, "IN_PROGRESS", agentID); err != nil {
				logger.Log("ERROR", "Daemon", "Failed to update task to IN_PROGRESS", map[string]interface{}{
					"project": project,
					"task":    targetTask.ID,
					"error":   err.Error(),
				})
				return
			}

			// Spawn a worker
			agentMu.Lock()
			activeAgents[project]++
			agentMu.Unlock()

			go func(proj string, task TaskView, workerID string) {
				defer func() {
					agentMu.Lock()
					activeAgents[proj]--
					agentMu.Unlock()
				}()

				logger.Log("INFO", "Daemon", fmt.Sprintf("Agent %s executing task %s", workerID, task.ID), map[string]interface{}{
					"project": proj,
				})

				runner := swarm.GetRunner(proj, workerID, &task)
				runner.SetLogger(logger)

				if err := runner.Setup(&task); err != nil {
					logger.Log("ERROR", "Daemon", "Runner setup failed", map[string]interface{}{
						"agent": workerID,
						"error": err.Error(),
					})
					projectManager.UpdateTaskStatus(proj, task.ID, "FAILED", workerID)
					return
				}

				// Execute native Go runner
				result, err := runner.Execute("", &task)
				finalStatus := "DONE"
				if err != nil {
					finalStatus = "FAILED"
					logger.Log("ERROR", "Daemon", "Runner execution failed", map[string]interface{}{
						"agent": workerID,
						"error": err.Error(),
					})
				} else {
					logger.Log("INFO", "Daemon", "Runner execution completed", map[string]interface{}{
						"agent":  workerID,
						"result": result,
					})
				}

				// Final state transition
				if err := projectManager.UpdateTaskStatus(proj, task.ID, finalStatus, workerID); err != nil {
					logger.Log("ERROR", "Daemon", "Failed to update final task status", map[string]interface{}{
						"error": err.Error(),
					})
				}

				// Teardown if atomic
				if task.Behavior.LifeCycle == "Atomic" || task.Behavior.LifeCycle == "" {
					runner.Teardown(&task)
				}
			}(project, *targetTask, agentID)
		}
	}

	// Initial scan
	projects, _ := projectManager.ListProjects(false)
	for _, p := range projects {
		processProject(p)
	}

	// 5. Main Event Loop
	logger.Log("INFO", "Daemon", "Entering event loop", nil)

	// Periodic ticker as fallback and to discover new projects
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// If a project file or directory changed
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// Determine which project changed
				rel, err := filepath.Rel(dataDir, event.Name)
				if err == nil && rel != "." && rel != ".." {
					// rel might be "project-name" or "project-name/project.yaml"
					projectName := rel
					if strings.Contains(rel, string(os.PathSeparator)) {
						projectName = filepath.Dir(rel)
					}

					// If it's the root dataDir, we might need to add new watches
					if event.Name == dataDir {
						addProjectWatches()
					} else {
						// Only process if it looks like a project file change
						if strings.HasSuffix(event.Name, ".yaml") || event.Op&fsnotify.Create != 0 {
							processProject(projectName)
						}
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			logger.Log("ERROR", "Daemon", "Watcher error", map[string]interface{}{"error": err.Error()})
		case <-ticker.C:
			// Fallback scan and watch update
			addProjectWatches()
			activeProjects, _ := projectManager.ListProjects(false)
			for _, p := range activeProjects {
				processProject(p)
			}
		}
	}
}
