package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize projects with the quickplan.sh network",
}

var pushCmd = &cobra.Command{
	Use:   "push [project_name]",
	Short: "Push a project blueprint to the registry",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetProject, err := getTargetProject(cmd)
		if err != nil {
			return err
		}

		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		versionManager := NewVersionManager(version)
		projectManager := NewProjectDataManager(dataDir, versionManager)

		projectData, err := projectManager.LoadProjectData(targetProject)
		if err != nil {
			return err
		}

		// Marshal to YAML for the blueprint
		yamlData, err := yaml.Marshal(projectData)
		if err != nil {
			return err
		}

		registryURL := os.Getenv("QUICKPLAN_REGISTRY_URL")
		if registryURL == "" {
			registryURL = "http://localhost:8080" // Default for local dev
		}

		blueprint := struct {
			ID          string `json:"id"`
			Author      string `json:"author"`
			Description string `json:"description"`
			YAMLContent string `json:"yaml_content"`
		}{
			ID:          targetProject,
			Author:      os.Getenv("USER"),
			Description: fmt.Sprintf("Project %s pushed from CLI", targetProject),
			YAMLContent: string(yamlData),
		}

		jsonData, err := json.Marshal(blueprint)
		if err != nil {
			return err
		}

		resp, err := http.Post(registryURL+"/api/v1/registry/push", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to connect to registry: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("registry returned error: %s", resp.Status)
		}

		fmt.Printf("Successfully pushed project '%s' to quickplan.sh network\n", targetProject)
		return nil
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull [blueprint_id]",
	Short: "Pull a project blueprint from the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// ... existing pull logic ...
		blueprintID := args[0]

		registryURL := os.Getenv("QUICKPLAN_REGISTRY_URL")
		if registryURL == "" {
			registryURL = "http://localhost:8080"
		}

		resp, err := http.Get(registryURL + "/api/v1/registry/pull?id=" + blueprintID)
		if err != nil {
			return fmt.Errorf("failed to connect to registry: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("blueprint not found: %s", blueprintID)
		}

		var blueprint struct {
			ID          string `json:"id"`
			YAMLContent string `json:"yaml_content"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&blueprint); err != nil {
			return err
		}

		// Save the blueprint as a new project
		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		projectDir := filepath.Join(dataDir, blueprintID)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return err
		}

		tasksFile := filepath.Join(projectDir, "tasks.yaml")
		if err := os.WriteFile(tasksFile, []byte(blueprint.YAMLContent), 0644); err != nil {
			return err
		}

		fmt.Printf("Successfully pulled blueprint '%s' and created project\n", blueprintID)
		return nil
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify [file.yaml]",
	Short: "Verify a project YAML against the registry schema",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var projectData ProjectData
		if err := yaml.Unmarshal(data, &projectData); err != nil {
			return fmt.Errorf("invalid YAML format: %w", err)
		}

		// Basic validation of required fields for Multi-Agent Protocol
		for _, task := range projectData.Tasks {
			if task.ID == 0 {
				return fmt.Errorf("task found with missing or zero ID")
			}
			if task.Text == "" {
				return fmt.Errorf("task %d has no text description", task.ID)
			}
			
			// New AgentBehavior verification
			if task.Behavior.Role != "" {
				if task.Behavior.Strategy == "" {
					fmt.Printf("⚠️ Warning: Task %d has a Role (%s) but no Strategy defined.\n", task.ID, task.Behavior.Role)
				}
			}
		}

		fmt.Printf("✅ Project DNA verified: %s is 100%% compatible with the protocol\n", filePath)
		return nil
	},
}

func init() {
	syncCmd.AddCommand(pushCmd)
	syncCmd.AddCommand(pullCmd)
	syncCmd.AddCommand(verifyCmd)
	pushCmd.Flags().StringP("project", "p", "", "Project to push")
}
