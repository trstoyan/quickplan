package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/trstoyan/quickplan/internal/swarm"
)

// BackgroundRunner executes workers asynchronously.
type BackgroundRunner struct {
	Logger         *swarm.EventLogger
	ProjectManager *ProjectDataManager
}

type executionPlan struct {
	Command    string
	PluginName string
}

// RunTask executes a single task synchronously.
func (br *BackgroundRunner) RunTask(project, agentID string, task *TaskView) error {
	plan, err := resolveTaskExecution(task)
	if err != nil {
		br.logExecutionError(agentID, "Task has no execution contract", err, "")
		_ = br.finalizeTask(project, agentID, task, "FAILED", err.Error())
		return err
	}

	var (
		output string
		runErr error
	)

	if plan.PluginName != "" {
		output, runErr = executePluginForTask(task, plan.PluginName)
	} else {
		runner := swarm.GetRunner(project, agentID, task)
		if br.Logger != nil {
			runner.SetLogger(br.Logger)
		}

		if err := runner.Setup(task); err != nil {
			runErr = fmt.Errorf("runner setup failed: %w", err)
		} else {
			output, runErr = runner.Execute(plan.Command, task)
		}

		// Teardown if it's an atomic lifecycle.
		if task.Behavior.LifeCycle == "Atomic" || task.Behavior.LifeCycle == "" {
			_ = runner.Teardown(task)
		}
	}

	finalStatus := "DONE"
	failureReason := ""
	if runErr != nil {
		finalStatus = "FAILED"
		failureReason = runErr.Error()
		br.logExecutionError(agentID, "Task execution failed", runErr, output)
	}

	if statusErr := br.finalizeTask(project, agentID, task, finalStatus, failureReason); statusErr != nil {
		if runErr == nil {
			return statusErr
		}
		br.logExecutionError(agentID, "Failed to persist task status after execution error", statusErr, "")
	}

	return runErr
}

// Start starts a worker agent asynchronously.
func (br *BackgroundRunner) Start(project, agentID string, task *TaskView) error {
	go func() {
		_ = br.RunTask(project, agentID, task)
	}()
	return nil
}

func (br *BackgroundRunner) finalizeTask(project, agentID string, task *TaskView, finalStatus, failureReason string) error {
	if task == nil || task.ID == "default" || br.ProjectManager == nil {
		return nil
	}

	if err := br.ProjectManager.UpdateTaskStatus(project, task.ID, finalStatus, agentID); err != nil {
		return err
	}

	if finalStatus == "FAILED" {
		if failureReason == "" {
			failureReason = "task execution failed"
		}
		if _, retryErr := br.ProjectManager.ScheduleRetryIfAllowed(project, task.ID, agentID, failureReason); retryErr != nil {
			return retryErr
		}
	}
	return nil
}

func (br *BackgroundRunner) logExecutionError(agentID, message string, err error, output string) {
	if br.Logger != nil {
		fields := map[string]interface{}{
			"error": err.Error(),
			"agent": agentID,
		}
		if output != "" {
			fields["output"] = output
		}
		br.Logger.Log("ERROR", "Swarm", message, fields)
		return
	}
	if output == "" {
		fmt.Printf("❌ %s for %s: %v\n", message, agentID, err)
		return
	}
	fmt.Printf("❌ %s for %s: %v\nOutput: %s\n", message, agentID, err, output)
}

var swarmCmd = &cobra.Command{
	Use:   "swarm",
	Short: "Orchestrate a swarm of AI agents",
}

var swarmStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a swarm of worker agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		workers, _ := cmd.Flags().GetInt("workers")
		projectName, _ := cmd.Flags().GetString("project")

		if projectName == "" {
			var err error
			projectName, err = getCurrentProject()
			if err != nil {
				return fmt.Errorf("could not determine project: %w", err)
			}
		}

		// 1. Initialize Logger
		dataDir, _ := getDataDir()
		logPath := filepath.Join(dataDir, "events.jsonl")
		logger, err := swarm.NewEventLogger(logPath)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer logger.Close()
		logger.OutputJSON = globalJSON

		// 2. Setup Bridge Directory
		bridgeDir := "/tmp"
		if _, err := os.Stat(bridgeDir); os.IsNotExist(err) {
			return fmt.Errorf("system bridge directory %s does not exist", bridgeDir)
		}

		// 3. Load Project manager
		projectManager := NewProjectDataManager(dataDir, NewVersionManager(version))

		// 4. Start Workers
		runner := &BackgroundRunner{
			Logger:         logger,
			ProjectManager: projectManager,
		}

		if !globalJSON {
			fmt.Printf("Initializing Swarm for project '%s' with %d workers...\n", projectName, workers)
		}

		pollInterval, _ := cmd.Flags().GetDuration("poll-interval")
		maxIdle, _ := cmd.Flags().GetDuration("max-idle")

		if workers < 1 {
			return fmt.Errorf("workers must be >= 1")
		}
		if pollInterval <= 0 {
			return fmt.Errorf("poll-interval must be > 0")
		}
		if maxIdle <= 0 {
			return fmt.Errorf("max-idle must be > 0")
		}
		if err := validateProjectExecutionContracts(projectManager, projectName); err != nil {
			return err
		}

		// 5. Supervisor Loop (if enabled)
		supervisorEnabled, _ := cmd.Flags().GetBool("supervisor")
		if supervisorEnabled {
			if !globalJSON {
				fmt.Println("🛡️ Supervisor active. Monitoring for blocked agents...")
			}
			go runSupervisor(projectName, logger)
		}

		if err := runSwarmToCompletion(projectName, workers, pollInterval, maxIdle, runner, projectManager, logger); err != nil {
			return err
		}

		if globalJSON {
			logger.Log("INFO", "Swarm", "Swarm run completed", map[string]interface{}{"workers": workers})
		} else {
			fmt.Println("Swarm run completed.")
		}

		return nil
	},
}

func runSwarmToCompletion(projectName string, workers int, pollInterval, maxIdle time.Duration, runner *BackgroundRunner, projectManager *ProjectDataManager, logger *swarm.EventLogger) error {
	if workers < 1 {
		return fmt.Errorf("workers must be >= 1")
	}

	var (
		wg           sync.WaitGroup
		stopCh       = make(chan struct{})
		stopOnce     sync.Once
		lastProgMu   sync.RWMutex
		lastProgAt   = time.Now()
		errMu        sync.Mutex
		executionErr error
	)

	stop := func() {
		stopOnce.Do(func() {
			close(stopCh)
		})
	}
	markProgress := func() {
		lastProgMu.Lock()
		lastProgAt = time.Now()
		lastProgMu.Unlock()
	}
	getLastProgress := func() time.Time {
		lastProgMu.RLock()
		defer lastProgMu.RUnlock()
		return lastProgAt
	}
	setExecutionErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if executionErr == nil {
			executionErr = err
			stop()
		}
		errMu.Unlock()
	}

	for i := 1; i <= workers; i++ {
		agentID := fmt.Sprintf("worker-%d", i)
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()

			for {
				select {
				case <-stopCh:
					return
				default:
				}

				task, claimErr := projectManager.ClaimNextRunnableTask(projectName, workerID)
				if claimErr != nil {
					if logger != nil {
						logger.Log("ERROR", "Swarm", "Failed to claim task", map[string]interface{}{
							"agent": workerID,
							"error": claimErr.Error(),
						})
					}
					time.Sleep(pollInterval)
					continue
				}

				if task == nil {
					snapshot, snapErr := projectManager.GetExecutionSnapshot(projectName)
					if snapErr != nil {
						if logger != nil {
							logger.Log("ERROR", "Swarm", "Failed to read execution snapshot", map[string]interface{}{
								"agent": workerID,
								"error": snapErr.Error(),
							})
						}
						time.Sleep(pollInterval)
						continue
					}

					if snapshot.AllTerminal {
						stop()
						return
					}

					if snapshot.InProgress == 0 && snapshot.Retrying == 0 && time.Since(getLastProgress()) >= maxIdle {
						setExecutionErr(fmt.Errorf("swarm stalled after %s (%s)", maxIdle, snapshot.Summary()))
						return
					}

					time.Sleep(pollInterval)
					continue
				}

				markProgress()
				runErr := runner.RunTask(projectName, workerID, task)
				if runErr != nil && logger != nil {
					logger.Log("WARN", "Swarm", "Worker execution failed", map[string]interface{}{
						"agent": workerID,
						"task":  task.ID,
						"error": runErr.Error(),
					})
				}
				markProgress()
			}
		}(agentID)
	}

	wg.Wait()

	errMu.Lock()
	defer errMu.Unlock()
	return executionErr
}

func validateProjectExecutionContracts(projectManager *ProjectDataManager, projectName string) error {
	views, _, err := projectManager.GetTaskViews(projectName)
	if err != nil {
		return err
	}

	var missing []string
	for _, task := range views {
		status := canonicalStatus(task.Status)
		if status == "DONE" || status == "FAILED" || status == "CANCELLED" {
			continue
		}
		if _, err := resolveTaskExecution(&task); err != nil {
			missing = append(missing, task.ID)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("project has tasks without execution contract (set behavior.command or plugin assignment): %s", strings.Join(missing, ", "))
	}
	return nil
}

func resolveTaskExecution(task *TaskView) (executionPlan, error) {
	if task == nil {
		return executionPlan{}, fmt.Errorf("nil task")
	}

	pluginName := strings.TrimSpace(task.Behavior.Plugin)
	if pluginName == "" {
		assigned := strings.TrimSpace(task.AssignedTo)
		if strings.HasPrefix(assigned, "plugin:") {
			pluginName = strings.TrimSpace(strings.TrimPrefix(assigned, "plugin:"))
		}
	}
	if pluginName != "" {
		return executionPlan{PluginName: pluginName}, nil
	}

	command := strings.TrimSpace(task.Behavior.Command)
	if command != "" {
		return executionPlan{Command: command}, nil
	}

	return executionPlan{}, fmt.Errorf("task %s has no execution contract", task.ID)
}

func executePluginForTask(task *TaskView, pluginName string) (string, error) {
	req := PluginRequest{
		TaskID:       task.ID,
		Role:         task.Behavior.Role,
		Strategy:     task.Behavior.Strategy,
		AllowedPaths: collectAllowedPaths(task),
	}

	resp, err := ExecutePlugin(pluginName, req)
	if err != nil {
		return "", err
	}

	status := strings.ToUpper(strings.TrimSpace(resp.Status))
	if status == "DONE" || status == "SUCCESS" || status == "OK" {
		return strings.TrimSpace(resp.Message), nil
	}
	if status == "" {
		status = "UNKNOWN"
	}
	return strings.TrimSpace(resp.Message), fmt.Errorf("plugin returned non-success status: %s", status)
}

func collectAllowedPaths(task *TaskView) []string {
	seen := make(map[string]struct{})
	var out []string

	appendPath := func(p string) {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}

	appendPath(task.WatchPath)
	for _, p := range task.WatchPaths {
		appendPath(p)
	}
	for _, p := range task.RequiresFiles {
		appendPath(p)
	}

	return out
}

func runSupervisor(projectName string, logger *swarm.EventLogger) {
	dataDir, _ := getDataDir()
	versionManager := NewVersionManager(version)
	projectManager := NewProjectDataManager(dataDir, versionManager)

	// Determine which file to watch
	taskFile := filepath.Join(dataDir, projectName, "project.yaml")
	if _, err := os.Stat(taskFile); os.IsNotExist(err) {
		taskFile = filepath.Join(dataDir, projectName, "tasks.yaml")
	}

	if logger != nil {
		logger.Log("INFO", "Supervisor", fmt.Sprintf("Watching %s for state transitions", taskFile), nil)
	} else {
		fmt.Printf("🛡️ Supervisor: Watching %s for state transitions...\n", taskFile)
	}

	for {
		views, _, err := projectManager.GetTaskViews(projectName)
		if err == nil {
			for _, task := range views {
				if task.Status == "BLOCKED" {
					if logger != nil {
						logger.Log("INFO", "Supervisor", fmt.Sprintf("Handling BLOCKED Task %s", task.ID), nil)
					} else {
						fmt.Printf("🛡️ Supervisor: Handling BLOCKED Task %s\n", task.ID)
					}

					// 1. Generate Remedy
					healTaskText := fmt.Sprintf("REMEDY: Resolve blocker in Task %s", task.ID)

					// 2. Inject (v1.1 or legacy handled by manager)
					if task.IsV11 {
						v11, _ := projectManager.LoadProjectV11(projectName)
						v11.Tasks = append(v11.Tasks, TaskV11{
							ID:     fmt.Sprintf("remedy-%d", time.Now().Unix()),
							Name:   healTaskText,
							Status: "TODO",
							Behavior: AgentBehavior{
								Role: "Senior Troubleshooter",
							},
							UpdatedAt: time.Now(),
						})
						projectManager.SaveProjectV11(projectName, v11)
					} else {
						legacy, _ := projectManager.LoadProjectData(projectName)
						maxID := 0
						for _, t := range legacy.Tasks {
							if t.ID > maxID {
								maxID = t.ID
							}
						}
						legacy.Tasks = append(legacy.Tasks, Task{
							ID:       maxID + 1,
							Text:     healTaskText,
							Created:  time.Now(),
							Behavior: AgentBehavior{Role: "Senior Troubleshooter"},
						})
						projectManager.SaveProjectData(projectName, legacy)
					}
					if logger != nil {
						logger.Log("INFO", "Supervisor", fmt.Sprintf("Injected remedy for %s", task.ID), nil)
					} else {
						fmt.Printf("🛡️ Supervisor: Injected remedy for %s\n", task.ID)
					}
				}
			}
		}

		// Wait for file change instead of fixed 10s sleep
		exec.Command("inotifywait", "-q", "-e", "modify", taskFile).Run()
		// Small cooldown to prevent rapid fire
		time.Sleep(500 * time.Millisecond)
	}
}

func init() {
	swarmCmd.AddCommand(swarmStartCmd)
	swarmStartCmd.Flags().IntP("workers", "w", 3, "Number of worker agents to spawn")
	swarmStartCmd.Flags().StringP("project", "p", "", "Project name")
	swarmStartCmd.Flags().Bool("supervisor", false, "Enable the Self-Healing Supervisor")
	swarmStartCmd.Flags().Duration("poll-interval", 500*time.Millisecond, "Polling interval for worker scheduling")
	swarmStartCmd.Flags().Duration("max-idle", 30*time.Second, "Maximum idle time before reporting a stalled swarm")
}
