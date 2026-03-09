package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
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

func TestRevisionBlobJSONContract(t *testing.T) {
	blob := RevisionBlob{
		Header: RevisionHeader{
			ProjectID:      "proj-1",
			RevID:          "rev-1",
			Alg:            "aes-256-gcm+x25519+ed25519:v1",
			AuthorPubKey:   "author-pub",
			CreatedAt:      time.Date(2026, time.March, 9, 10, 11, 12, 0, time.UTC),
			CiphertextHash: "abc123",
		},
		NonceB64:      "nonce",
		CiphertextB64: "ciphertext",
		SignatureB64:  "signature",
	}

	data, err := json.Marshal(blob)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	expectedTopLevelKeys := []string{"ciphertext_b64", "header", "nonce_b64", "signature_b64"}
	actualTopLevelKeys := make([]string, 0, len(decoded))
	for key := range decoded {
		actualTopLevelKeys = append(actualTopLevelKeys, key)
	}
	sort.Strings(actualTopLevelKeys)
	if !reflect.DeepEqual(actualTopLevelKeys, expectedTopLevelKeys) {
		t.Fatalf("unexpected top-level keys: got %v want %v", actualTopLevelKeys, expectedTopLevelKeys)
	}

	header, ok := decoded["header"].(map[string]any)
	if !ok {
		t.Fatalf("header should be an object, got %T", decoded["header"])
	}

	expectedHeaderKeys := []string{"alg", "author_pubkey", "ciphertext_hash", "created_at", "project_id", "rev_id"}
	actualHeaderKeys := make([]string, 0, len(header))
	for key := range header {
		actualHeaderKeys = append(actualHeaderKeys, key)
	}
	sort.Strings(actualHeaderKeys)
	if !reflect.DeepEqual(actualHeaderKeys, expectedHeaderKeys) {
		t.Fatalf("unexpected header keys: got %v want %v", actualHeaderKeys, expectedHeaderKeys)
	}
}
