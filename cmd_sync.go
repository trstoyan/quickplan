package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize projects with a compatible remote service",
}

var pushCmd = &cobra.Command{
	Use:   "push [project_name]",
	Short: "Push a project blueprint to a remote registry",
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

		blueprintFormat := "legacy"
		blueprintSchemaVersion := ""
		var yamlData []byte

		if v11, err := projectManager.LoadProjectV11(targetProject); err == nil {
			blueprintFormat = "v1.1"
			blueprintSchemaVersion = "1.1"
			yamlData, err = yaml.Marshal(v11)
			if err != nil {
				return err
			}
		} else {
			projectData, err := projectManager.LoadProjectData(targetProject)
			if err != nil {
				return err
			}
			yamlData, err = yaml.Marshal(projectData)
			if err != nil {
				return err
			}
		}

		blueprintVersion, _ := cmd.Flags().GetString("version")
		if strings.TrimSpace(blueprintVersion) == "" {
			blueprintVersion = "1"
		}

		registryURL := os.Getenv("QUICKPLAN_REGISTRY_URL")
		if registryURL == "" {
			registryURL = "http://localhost:8081" // Default for local dev
		}

		blueprint := struct {
			ID            string `json:"id"`
			Author        string `json:"author"`
			Description   string `json:"description"`
			YAMLContent   string `json:"yaml_content"`
			Format        string `json:"format,omitempty"`
			SchemaVersion string `json:"schema_version,omitempty"`
			Version       string `json:"version,omitempty"`
		}{
			ID:            targetProject,
			Author:        os.Getenv("USER"),
			Description:   fmt.Sprintf("Project %s pushed from CLI", targetProject),
			YAMLContent:   string(yamlData),
			Format:        blueprintFormat,
			SchemaVersion: blueprintSchemaVersion,
			Version:       blueprintVersion,
		}

		jsonData, err := json.Marshal(blueprint)
		if err != nil {
			return err
		}

		req, err := http.NewRequest(http.MethodPost, registryURL+"/api/v1/registry/push", bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		applyWebAuth(req)

		resp, err := newWebClient(15 * time.Second).Do(req)
		if err != nil {
			return fmt.Errorf("failed to connect to registry: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("registry returned error: %s", resp.Status)
		}

			fmt.Printf("Successfully pushed project '%s' to the remote service\n", targetProject)
			return nil
		},
	}

var pullCmd = &cobra.Command{
	Use:   "pull [blueprint_id]",
	Short: "Pull a project blueprint from a remote registry",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFlag, _ := cmd.Flags().GetString("project")
		blueprintID := ""
		if len(args) > 0 {
			blueprintID = args[0]
		} else if projectFlag != "" {
			blueprintID = projectFlag
		}
		if blueprintID == "" {
			return fmt.Errorf("blueprint ID is required")
		}

		localProjectName := projectFlag
		if localProjectName == "" {
			localProjectName = blueprintID
		}

		registryURL := os.Getenv("QUICKPLAN_REGISTRY_URL")
		if registryURL == "" {
			registryURL = "http://localhost:8081"
		}

		req, err := http.NewRequest(http.MethodGet, registryURL+"/api/v1/registry/pull?id="+blueprintID, nil)
		if err != nil {
			return err
		}
		applyWebAuth(req)

		resp, err := newWebClient(15 * time.Second).Do(req)
		if err != nil {
			return fmt.Errorf("failed to connect to registry: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("blueprint not found: %s", blueprintID)
		}

		var blueprint struct {
			ID            string `json:"id"`
			YAMLContent   string `json:"yaml_content"`
			Format        string `json:"format"`
			SchemaVersion string `json:"schema_version"`
			Version       string `json:"version"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&blueprint); err != nil {
			return err
		}

		// Save the blueprint as a new project
		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		projectDir := filepath.Join(dataDir, localProjectName)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return err
		}

		targetFile := filepath.Join(projectDir, "tasks.yaml")
		format := strings.ToLower(strings.TrimSpace(blueprint.Format))
		schemaVersion := strings.TrimSpace(blueprint.SchemaVersion)
		content := strings.TrimSpace(blueprint.YAMLContent)
		if schemaVersion == "1.1" || format == "v1.1" || strings.Contains(content, "schema_version: \"1.1\"") || strings.Contains(content, "schema_version: '1.1'") || strings.Contains(content, "schema_version: 1.1") {
			targetFile = filepath.Join(projectDir, "project.yaml")
		}

		if err := os.WriteFile(targetFile, []byte(blueprint.YAMLContent), 0644); err != nil {
			return err
		}

		fmt.Printf("Successfully pulled blueprint '%s' (v%s) into project '%s'\n", blueprintID, blueprint.Version, localProjectName)
		return nil
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify [file.yaml]",
	Short: "Verify a project YAML against the blueprint schema",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var schemaProbe struct {
			SchemaVersion string `yaml:"schema_version"`
		}
		if err := yaml.Unmarshal(data, &schemaProbe); err != nil {
			return fmt.Errorf("invalid YAML format: %w", err)
		}

		if schemaProbe.SchemaVersion == "1.1" {
			var projectV11 ProjectV11
			if err := yaml.Unmarshal(data, &projectV11); err != nil {
				return fmt.Errorf("invalid v1.1 YAML format: %w", err)
			}
			if err := ValidateProjectV11(&projectV11); err != nil {
				return fmt.Errorf("v1.1 validation failed: %w", err)
			}
			fmt.Printf("✅ Project DNA verified: %s is v1.1 protocol compatible\n", filePath)
			return nil
		}

		var projectData ProjectData
		if err := yaml.Unmarshal(data, &projectData); err != nil {
			return fmt.Errorf("invalid legacy YAML format: %w", err)
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
	pushCmd.Flags().String("version", "1", "Blueprint version for registry immutability")
	pullCmd.Flags().StringP("project", "p", "", "Local project name override (defaults to blueprint ID)")
}
