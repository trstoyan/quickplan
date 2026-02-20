package main

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/sumix/quickplan/pkg/crypto"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export project data as an encrypted blob",
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
		
		// 2. Load Project YAML
		projectFile := filepath.Join(projectDir, "project.yaml")
		if _, err := os.Stat(projectFile); os.IsNotExist(err) {
			projectFile = filepath.Join(projectDir, "tasks.yaml") // legacy fallback
		}
		
		plaintext, err := os.ReadFile(projectFile)
		if err != nil {
			return fmt.Errorf("failed to read project data: %w", err)
		}

		// 3. Load Identity
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

		// 4. Get Project Key
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
			return fmt.Errorf("encryption not initialized for this project (run 'project-key init'): %w", err)
		}

		// 5. Build Header
		// Ensure stable project ID
		projectIDPath := filepath.Join(projectDir, ".qp_crypto", "project.id")
		projectID, err := os.ReadFile(projectIDPath)
		if err != nil {
			projectID = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
			os.WriteFile(projectIDPath, projectID, 0600)
		}

		revID := fmt.Sprintf("%d", time.Now().UnixNano())
		
		header := crypto.RevisionHeader{
			ProjectID:    string(projectID),
			RevID:        revID,
			Alg:          "aes-256-gcm+x25519+ed25519:v1",
			AuthorPubKey: identity.Ed25519Pub,
			CreatedAt:    time.Now(),
		}

		// 6. Encrypt
		headerJSON, err := json.Marshal(header)
		if err != nil {
			return err
		}
		nonce, ciphertext, err := crypto.Encrypt(projectKey, plaintext, headerJSON)
		if err != nil {
			return err
		}

		hash := sha256.Sum256(ciphertext)
		header.CiphertextHash = hex.EncodeToString(hash[:])

		// 7. Sign
		edPrivBytes, err := base64.StdEncoding.DecodeString(identity.Ed25519Priv)
		if err != nil {
			return err
		}
		edPriv := ed25519.PrivateKey(edPrivBytes)

		blob := crypto.RevisionBlob{
			Header:        header,
			NonceB64:      base64.StdEncoding.EncodeToString(nonce),
			CiphertextB64: base64.StdEncoding.EncodeToString(ciphertext),
		}
		if err := blob.Sign(edPriv); err != nil {
			return err
		}

		// 8. Output
		outPath, _ := cmd.Flags().GetString("out")
		if outPath == "" {
			outPath = fmt.Sprintf("%s_%s.qpblob.json", projectName, revID)
		}

		blobData, err := json.MarshalIndent(blob, "", "  ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(outPath, blobData, 0644); err != nil {
			return err
		}

		fmt.Printf("✓ Project exported to %s\n", outPath)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringP("project", "p", "", "Project name")
	exportCmd.Flags().String("identity", "", "Path to identity.json")
	exportCmd.Flags().String("out", "", "Output filename")
}
