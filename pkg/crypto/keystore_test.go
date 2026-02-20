package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestKeystore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qp-keystore-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	userPriv, _ := GenerateX25519()
	userPub := userPriv.PublicKey()

	// 1. Init
	key1, err := InitProjectKey(tmpDir, userPub, userPriv)
	if err != nil {
		t.Fatalf("InitProjectKey failed: %v", err)
	}

	// 2. Get
	key2, err := GetProjectKey(tmpDir, userPriv, userPub)
	if err != nil {
		t.Fatalf("GetProjectKey failed: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Error("keys do not match")
	}

	// 3. Test missing files
	os.Remove(filepath.Join(tmpDir, CryptoDir, ProjectKeyFile))
	_, err = GetProjectKey(tmpDir, userPriv, userPub)
	if err == nil {
		t.Error("GetProjectKey should fail if key file is missing")
	}
}
