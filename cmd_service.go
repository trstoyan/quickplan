package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

// Service Template
const serviceTemplate = `[Unit]
Description=QuickPlan Swarm Daemon
After=network.target

[Service]
Type=simple
ExecStart={{.ExecutablePath}} daemon
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the QuickPlan systemd background service",
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and start the systemd user service",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Get User Config Dir
		usr, err := user.Current()
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		configDir := filepath.Join(usr.HomeDir, ".config", "systemd", "user")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create systemd directory: %w", err)
		}

		// 2. Resolve Binary Path
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine executable path: %w", err)
		}
		// Ensure absolute path
		execPath, err = filepath.Abs(execPath)
		if err != nil {
			return err
		}

		// 3. Write Service File
		serviceFile := filepath.Join(configDir, "quickplan.service")
		f, err := os.Create(serviceFile)
		if err != nil {
			return fmt.Errorf("failed to create service file: %w", err)
		}
		defer f.Close()

		tmpl, err := template.New("service").Parse(serviceTemplate)
		if err != nil {
			return err
		}

		data := struct {
			ExecutablePath string
		}{
			ExecutablePath: execPath,
		}

		if err := tmpl.Execute(f, data); err != nil {
			return err
		}

		fmt.Printf("Created service file: %s\n", serviceFile)

		// 4. Reload and Enable
		cmds := [][]string{
			{"systemctl", "--user", "daemon-reload"},
			{"systemctl", "--user", "enable", "--now", "quickplan.service"},
		}

		for _, c := range cmds {
			cmd := exec.Command(c[0], c[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to run %v: %w", c, err)
			}
		}

		fmt.Println("QuickPlan background service installed and started.")
		return nil
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Stop and remove the systemd user service",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Disable and Stop
		_ = exec.Command("systemctl", "--user", "disable", "--now", "quickplan.service").Run()

		// 2. Remove File
		usr, _ := user.Current()
		serviceFile := filepath.Join(usr.HomeDir, ".config", "systemd", "user", "quickplan.service")
		if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove service file: %w", err)
		}

		// 3. Reload
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()

		fmt.Println("QuickPlan background service uninstalled.")
		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	rootCmd.AddCommand(serviceCmd)
}
