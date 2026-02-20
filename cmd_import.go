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

var importCmd = &cobra.Command{
	Use:   "import <blobfile>",
	Short: "Import a project from an encrypted blob",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		blobPath := args[0]
		
		// 1. Load Blob
		blobData, err := os.ReadFile(blobPath)
		if err != nil {
			return fmt.Errorf("failed to read blob: %w", err)
		}
		var blob crypto.RevisionBlob
		if err := json.Unmarshal(blobData, &blob); err != nil {
			return fmt.Errorf("invalid blob format: %w", err)
		}

		// 2. Verify Signature
		if err := blob.Verify(); err != nil {
			return fmt.Errorf("integrity/signature check failed: %w", err)
		}

		// 3. Resolve Project
		projectName, _ := cmd.Flags().GetString("project")
		if projectName == "" {
			projectName = blob.Header.ProjectID // Use project ID as name by default
		}

		dataDir, err := getDataDir()
		if err != nil {
			return err
		}
		projectDir := filepath.Join(dataDir, projectName)
		
		// Ensure project exists or create it
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				return err
			}
		}

		// 4. Load Identity
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
			return err
		}

		// 5. Get Project Key
		xPrivBytes, err := base64.StdEncoding.DecodeString(identity.X25519Priv)
		if err != nil {
			return err
		}
		xPubBytes, err := base64.StdEncoding.DecodeString(identity.X25519Pub)
		if err != nil {
			return err
		}
		xPriv, err := ecdh.X25519().NewPrivateKey(xPrivBytes)
		if err != nil {
			return err
		}
		xPub, err := ecdh.X25519().NewPublicKey(xPubBytes)
		if err != nil {
			return err
		}

		projectKey, err := crypto.GetProjectKey(projectDir, xPriv, xPub)
		if err != nil {
			return fmt.Errorf("encryption key missing for this project: %w", err)
		}

		// 6. Decrypt
		nonce, err := base64.StdEncoding.DecodeString(blob.NonceB64)
		if err != nil {
			return err
		}
		ciphertext, err := base64.StdEncoding.DecodeString(blob.CiphertextB64)
		if err != nil {
			return err
		}

		// Match the AAD used during export (header with empty CiphertextHash)
		aadHeader := blob.Header
		aadHeader.CiphertextHash = ""
		headerJSON, err := json.Marshal(aadHeader)
		if err != nil {
			return err
		}

		plaintext, err := crypto.Decrypt(projectKey, nonce, ciphertext, headerJSON)
		if err != nil {
			return fmt.Errorf("decryption failed: %w", err)
		}

		// 7. Atomic Write
		projectFile := filepath.Join(projectDir, "project.yaml")
		tmpFile := projectFile + ".tmp"
		if err := os.WriteFile(tmpFile, plaintext, 0644); err != nil {
			return err
		}
		if err := os.Rename(tmpFile, projectFile); err != nil {
			return err
		}

		fmt.Printf("✓ Project '%s' imported successfully\n", projectName)
		return nil
	},
}

func init() {
	importCmd.Flags().StringP("project", "p", "", "Target project name")
	importCmd.Flags().String("identity", "", "Path to identity.json")
}
