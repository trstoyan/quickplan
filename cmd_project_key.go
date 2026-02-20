package main

import (
	"crypto/ecdh"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sumix/quickplan/pkg/crypto"
)

var projectKeyCmd = &cobra.Command{
	Use:   "project-key",
	Short: "Manage project cryptographic keys",
}

var projectKeyInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new project symmetric key",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName, _ := cmd.Flags().GetString("project")
		if projectName == "" {
			var err error
			projectName, err = getCurrentProject()
			if err != nil {
				return err
			}
		}

		// 1. Resolve project directory
		dataDir, err := getDataDir()
		if err != nil {
			return err
		}
		projectDir := filepath.Join(dataDir, projectName)
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			return fmt.Errorf("project '%s' does not exist", projectName)
		}

		// 2. Load Identity
		identityPath, _ := cmd.Flags().GetString("identity")
		if identityPath == "" {
			home, _ := os.UserHomeDir()
			identityPath = filepath.Join(home, ".config", "quickplan", "identity.json")
		}

		identityData, err := os.ReadFile(identityPath)
		if err != nil {
			return fmt.Errorf("failed to read identity: %w", err)
		}

		var identity UserIdentity
		if err := json.Unmarshal(identityData, &identity); err != nil {
			return fmt.Errorf("invalid identity format: %w", err)
		}

		// Parse X25519 keys
		xPubBytes, err := base64.StdEncoding.DecodeString(identity.X25519Pub)
		if err != nil {
			return err
		}
		xPrivBytes, err := base64.StdEncoding.DecodeString(identity.X25519Priv)
		if err != nil {
			return err
		}

		xPub, err := ecdh.X25519().NewPublicKey(xPubBytes)
		if err != nil {
			return err
		}
		xPriv, err := ecdh.X25519().NewPrivateKey(xPrivBytes)
		if err != nil {
			return err
		}

		// 3. Init Project Key
		_, err = crypto.InitProjectKey(projectDir, xPub, xPriv)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Cryptographic keystore initialized for project '%s'\n", projectName)
		return nil
	},
}

func init() {
	projectKeyCmd.AddCommand(projectKeyInitCmd)
	projectKeyInitCmd.Flags().StringP("project", "p", "", "Project name")
	projectKeyInitCmd.Flags().String("identity", "", "Path to identity.json")
}
