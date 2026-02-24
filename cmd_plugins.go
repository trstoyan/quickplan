package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage agent plugins",
}

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed agent plugins",
	RunE: func(cmd *cobra.Command, args []string) error {
		plugins, err := ListPlugins()
		if err != nil {
			return err
		}

		if len(plugins) == 0 {
			fmt.Println("No plugins found in ~/.quickplan/plugins/")
			return nil
		}

		fmt.Println("Installed Plugins:")
		for _, p := range plugins {
			fmt.Printf("  - %s (%s)\n", p.Name, p.Path)
		}
		return nil
	},
}

type PluginInfo struct {
	Name string
	Path string
}

func getPluginsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".quickplan", "plugins")
}

func ListPlugins() ([]PluginInfo, error) {
	dir := getPluginsDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var plugins []PluginInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			plugins = append(plugins, PluginInfo{
				Name: entry.Name(),
				Path: filepath.Join(dir, entry.Name()),
			})
		}
	}
	return plugins, nil
}

type PluginRequest struct {
	TaskID       string   `json:"task_id"`
	Role         string   `json:"role"`
	Strategy     string   `json:"strategy"`
	AllowedPaths []string `json:"allowed_paths"`
}

type PluginResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func ExecutePlugin(pluginName string, req PluginRequest) (*PluginResponse, error) {
	path := filepath.Join(getPluginsDir(), pluginName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin %s not found", pluginName)
	}

	input, _ := json.Marshal(req)
	cmd := exec.Command(path)
	cmd.Stdin = bytes.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("plugin execution failed: %w\nStderr: %s", err, stderr.String())
	}

	var resp PluginResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse plugin response: %w", err)
	}

	return &resp, nil
}

func init() {
	pluginsCmd.AddCommand(pluginsListCmd)
}
