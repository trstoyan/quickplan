package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/trstoyan/quickplan/pkg/crypto"
)

type UserIdentity struct {
	Ed25519Pub  string `json:"ed25519_pub"`
	Ed25519Priv string `json:"ed25519_priv"`
	X25519Pub   string `json:"x25519_pub"`
	X25519Priv  string `json:"x25519_priv"`
}

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate a new user identity for encryption",
	RunE: func(cmd *cobra.Command, args []string) error {
		outPath, _ := cmd.Flags().GetString("out")
		if outPath == "" {
			home, _ := os.UserHomeDir()
			outPath = filepath.Join(home, ".config", "quickplan", "identity.json")
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0700); err != nil {
			return err
		}

		// Check if exists
		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("identity file already exists at %s", outPath)
		}

		// 1. Ed25519
		edPub, edPriv, err := crypto.GenerateEd25519()
		if err != nil {
			return err
		}

		// 2. X25519
		xPriv, err := crypto.GenerateX25519()
		if err != nil {
			return err
		}
		xPub := xPriv.PublicKey()

		xPrivBytes := xPriv.Bytes()
		xPubBytes := xPub.Bytes()

		identity := UserIdentity{
			Ed25519Pub:  base64.StdEncoding.EncodeToString(edPub),
			Ed25519Priv: base64.StdEncoding.EncodeToString(edPriv),
			X25519Pub:   base64.StdEncoding.EncodeToString(xPubBytes),
			X25519Priv:  base64.StdEncoding.EncodeToString(xPrivBytes),
		}

		data, err := json.MarshalIndent(identity, "", "  ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(outPath, data, 0600); err != nil {
			return err
		}

		fmt.Printf("✓ Identity generated at %s\n", outPath)
		return nil
	},
}

func init() {
	keygenCmd.Flags().String("out", "", "Output path for identity.json")
}
