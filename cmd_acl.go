package main

import (
	"crypto/ecdh"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/trstoyan/quickplan/pkg/crypto"
	"gopkg.in/yaml.v3"
)

type ACLMember struct {
	Ed25519Pub          string `yaml:"ed25519_pub"`
	X25519Pub           string `yaml:"x25519_pub"`
	Role                string `yaml:"role"`
	WrappedProjectKey   string `yaml:"wrapped_project_key"`
	WrappedProjectNonce string `yaml:"wrapped_project_nonce"`
}

type ProjectACL struct {
	OwnerEd25519Pub string      `yaml:"owner_ed25519_pub"`
	Members         []ACLMember `yaml:"members"`
}

var aclCmd = &cobra.Command{
	Use:   "acl",
	Short: "Manage project access control list",
}

var aclAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a collaborator to the project ACL",
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
			return err
		}

		// 3. Get Project Key
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
			return err
		}

		// 4. Wrap Key for Collaborator
		collabXPubB64, _ := cmd.Flags().GetString("x25519-pub")
		collabEdPubB64, _ := cmd.Flags().GetString("ed25519-pub")
		role, _ := cmd.Flags().GetString("role")

		collabXPubBytes, err := base64.StdEncoding.DecodeString(collabXPubB64)
		if err != nil {
			return fmt.Errorf("invalid x25519-pub: %w", err)
		}
		collabXPub, err := ecdh.X25519().NewPublicKey(collabXPubBytes)
		if err != nil {
			return err
		}

		nonce, wrapped, err := crypto.WrapKeyX25519(projectKey, collabXPub, xPriv)
		if err != nil {
			return err
		}

		// 5. Update ACL
		aclPath := filepath.Join(projectDir, ".qp_crypto", "acl.yaml")
		var acl ProjectACL
		aclData, err := os.ReadFile(aclPath)
		if err == nil {
			yaml.Unmarshal(aclData, &acl)
		} else {
			acl.OwnerEd25519Pub = identity.Ed25519Pub
		}

		member := ACLMember{
			Ed25519Pub:          collabEdPubB64,
			X25519Pub:           collabXPubB64,
			Role:                role,
			WrappedProjectKey:   base64.StdEncoding.EncodeToString(wrapped),
			WrappedProjectNonce: base64.StdEncoding.EncodeToString(nonce),
		}
		acl.Members = append(acl.Members, member)

		newACLData, err := yaml.Marshal(acl)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(aclPath), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(aclPath, newACLData, 0600); err != nil {
			return err
		}

		fmt.Printf("✓ Added collaborator %s to ACL for project '%s'\n", collabEdPubB64[:12], projectName)
		return nil
	},
}

func init() {
	aclCmd.AddCommand(aclAddCmd)
	aclAddCmd.Flags().StringP("project", "p", "", "Project name")
	aclAddCmd.Flags().String("identity", "", "Path to identity.json")
	aclAddCmd.Flags().String("x25519-pub", "", "Collaborator X25519 public key (base64)")
	aclAddCmd.Flags().String("ed25519-pub", "", "Collaborator Ed25519 public key (base64)")
	aclAddCmd.Flags().String("role", "editor", "Collaborator role")
}
