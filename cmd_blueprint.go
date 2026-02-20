package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var blueprintCmd = &cobra.Command{
	Use:   "blueprint",
	Short: "Manage project blueprints and signatures",
}

var signCmd = &cobra.Command{
	Use:   "sign --key <private_key_file> <project.yaml>",
	Short: "Sign a project blueprint using Ed25519",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		yamlFile := args[0]
		keyPath, _ := cmd.Flags().GetString("key")
		if keyPath == "" {
			return fmt.Errorf("private key file (--key) is required")
		}

		// 1. Load Private Key
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key: %w", err)
		}
		if len(keyData) != ed25519.PrivateKeySize {
			return fmt.Errorf("invalid private key size: expected %d bytes", ed25519.PrivateKeySize)
		}
		privKey := ed25519.PrivateKey(keyData)

		// 2. Load and Hash YAML
		yamlData, err := os.ReadFile(yamlFile)
		if err != nil {
			return fmt.Errorf("failed to read yaml file: %w", err)
		}
		hash := sha256.Sum256(yamlData)

		// 3. Sign
		signature := ed25519.Sign(privKey, hash[:])
		pubKey := privKey.Public().(ed25519.PublicKey)

		// 4. Output
		fmt.Printf("File:      %s\n", yamlFile)
		fmt.Printf("SHA256:    %s\n", hex.EncodeToString(hash[:]))
		fmt.Printf("Signature: %s\n", base64.StdEncoding.EncodeToString(signature))
		fmt.Printf("PublicKey: %s\n", base64.StdEncoding.EncodeToString(pubKey))

		return nil
	},
}

var verifyBlueprintCmd = &cobra.Command{
	Use:   "verify <project.yaml>",
	Short: "Verify a blueprint signature",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		yamlFile := args[0]
		sigB64, _ := cmd.Flags().GetString("signature")
		pubB64, _ := cmd.Flags().GetString("public-key")

		if sigB64 == "" || pubB64 == "" {
			return fmt.Errorf("both --signature and --public-key are required for verification")
		}

		// 1. Decode inputs
		sig, err := base64.StdEncoding.DecodeString(sigB64)
		if err != nil {
			return fmt.Errorf("invalid signature encoding")
		}
		pub, err := base64.StdEncoding.DecodeString(pubB64)
		if err != nil || len(pub) != ed25519.PublicKeySize {
			return fmt.Errorf("invalid public key format")
		}

		// 2. Hash File
		yamlData, err := os.ReadFile(yamlFile)
		if err != nil {
			return fmt.Errorf("failed to read yaml file: %w", err)
		}
		hash := sha256.Sum256(yamlData)

		// 3. Verify
		if ed25519.Verify(pub, hash[:], sig) {
			fmt.Println("✅ Signature VALID")
		} else {
			return fmt.Errorf("❌ Signature INVALID")
		}

		return nil
	},
}

func init() {
	blueprintCmd.AddCommand(signCmd)
	signCmd.Flags().StringP("key", "k", "", "Path to Ed25519 private key file")
	
	blueprintCmd.AddCommand(verifyBlueprintCmd)
	verifyBlueprintCmd.Flags().String("signature", "", "Base64 signature")
	verifyBlueprintCmd.Flags().String("public-key", "", "Base64 public key")
}
