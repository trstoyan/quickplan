package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	CryptoDir          = ".qp_crypto"
	ProjectKeyFile     = "project.key.wrapped"
	ProjectKeyNonceFile = "project.key.nonce"
)

// InitProjectKey creates a random 32-byte project key, wraps it for the user, and stores it.
func InitProjectKey(projectDir string, userPub *ecdh.PublicKey, userPriv *ecdh.PrivateKey) ([]byte, error) {
	cryptoPath := filepath.Join(projectDir, CryptoDir)
	if err := os.MkdirAll(cryptoPath, 0700); err != nil {
		return nil, err
	}

	projectKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, projectKey); err != nil {
		return nil, err
	}

	nonce, wrapped, err := WrapKeyX25519(projectKey, userPub, userPriv)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(cryptoPath, ProjectKeyFile), []byte(base64.StdEncoding.EncodeToString(wrapped)), 0600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(cryptoPath, ProjectKeyNonceFile), []byte(base64.StdEncoding.EncodeToString(nonce)), 0600); err != nil {
		return nil, err
	}

	return projectKey, nil
}

// GetProjectKey unwraps the stored project key using the user's private key.
func GetProjectKey(projectDir string, userPriv *ecdh.PrivateKey, userPub *ecdh.PublicKey) ([]byte, error) {
	cryptoPath := filepath.Join(projectDir, CryptoDir)
	
	wrappedB64, err := os.ReadFile(filepath.Join(cryptoPath, ProjectKeyFile))
	if err != nil {
		return nil, fmt.Errorf("missing project key: %w", err)
	}
	nonceB64, err := os.ReadFile(filepath.Join(cryptoPath, ProjectKeyNonceFile))
	if err != nil {
		return nil, fmt.Errorf("missing project key nonce: %w", err)
	}

	wrapped, err := base64.StdEncoding.DecodeString(string(wrappedB64))
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(string(nonceB64))
	if err != nil {
		return nil, err
	}

	return UnwrapKeyX25519(nonce, wrapped, userPriv, userPub)
}
