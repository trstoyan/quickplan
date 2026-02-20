package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestBlobLifecycle(t *testing.T) {
	pub, priv, _ := GenerateEd25519()
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	ciphertext := []byte("encrypted content")
	ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)
	nonceB64 := base64.StdEncoding.EncodeToString(make([]byte, 12))

	blob := &RevisionBlob{
		Header: RevisionHeader{
			ProjectID:      "proj-123",
			RevID:          "rev-456",
			Alg:            "aes-256-gcm+ed25519",
			AuthorPubKey:   pubB64,
			CreatedAt:      time.Now().Round(time.Second),
			CiphertextHash: "3ac54cc3513dc0536ef06da060006de0606dbde6dfbd60606060606060606060", // dummy
		},
		NonceB64:      nonceB64,
		CiphertextB64: ciphertextB64,
	}

	// Update hash
	hash := sha256.Sum256(ciphertext)
	blob.Header.CiphertextHash = fmt.Sprintf("%x", hash)

	if err := blob.Sign(priv); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if err := blob.Verify(); err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	// Tamper with header
	blob.Header.ProjectID = "tampered"
	if err := blob.Verify(); err == nil {
		t.Error("Verify should have failed after header tamper")
	}
	blob.Header.ProjectID = "proj-123"

	// Tamper with ciphertext
	blob.CiphertextB64 = base64.StdEncoding.EncodeToString([]byte("tampered content"))
	if err := blob.Verify(); err == nil {
		t.Error("Verify should have failed after ciphertext tamper")
	}
}

func TestBlobEncoding(t *testing.T) {
	blob := &RevisionBlob{
		Header: RevisionHeader{
			ProjectID: "p1",
		},
		SignatureB64: "sig",
	}

	data, err := json.Marshal(blob)
	if err != nil {
		t.Fatal(err)
	}

	var decoded RevisionBlob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Header.ProjectID != "p1" {
		t.Errorf("encoding/decoding mismatch")
	}
}
