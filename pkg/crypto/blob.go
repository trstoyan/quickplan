package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// RevisionHeader contains metadata about an encrypted revision.
type RevisionHeader struct {
	ProjectID      string    `json:"project_id"`
	RevID          string    `json:"rev_id"`
	Alg            string    `json:"alg"`
	AuthorPubKey   string    `json:"author_pubkey"` // Base64 Ed25519 public key
	CreatedAt      time.Time `json:"created_at"`
	CiphertextHash string    `json:"ciphertext_hash"` // Hex SHA256 of ciphertext bytes
}

// RevisionBlob is the portable container for an encrypted project.yaml.
type RevisionBlob struct {
	Header       RevisionHeader `json:"header"`
	NonceB64     string         `json:"nonce_b64"`
	CiphertextB64 string         `json:"ciphertext_b64"`
	SignatureB64 string         `json:"signature_b64"`
}

// Sign signs the blob's header and ciphertext hash using the author's private key.
func (b *RevisionBlob) Sign(priv ed25519.PrivateKey) error {
	msg, err := b.canonicalBytes()
	if err != nil {
		return err
	}
	sig := ed25519.Sign(priv, msg)
	b.SignatureB64 = base64.StdEncoding.EncodeToString(sig)
	return nil
}

// Verify checks the blob's signature against the header and ciphertext.
func (b *RevisionBlob) Verify() error {
	pubBytes, err := base64.StdEncoding.DecodeString(b.Header.AuthorPubKey)
	if err != nil {
		return fmt.Errorf("invalid author pubkey encoding: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid author pubkey size")
	}
	pub := ed25519.PublicKey(pubBytes)

	sigBytes, err := base64.StdEncoding.DecodeString(b.SignatureB64)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	msg, err := b.canonicalBytes()
	if err != nil {
		return err
	}

	if !ed25519.Verify(pub, msg, sigBytes) {
		return fmt.Errorf("signature verification failed")
	}

	// Verify ciphertext hash
	ciphertext, err := base64.StdEncoding.DecodeString(b.CiphertextB64)
	if err != nil {
		return fmt.Errorf("invalid ciphertext encoding: %w", err)
	}
	hash := sha256.Sum256(ciphertext)
	if fmt.Sprintf("%x", hash) != b.Header.CiphertextHash {
		return fmt.Errorf("ciphertext hash mismatch")
	}

	return nil
}

func (b *RevisionBlob) canonicalBytes() ([]byte, error) {
	headerJSON, err := json.Marshal(b.Header)
	if err != nil {
		return nil, err
	}
	// We sign the header JSON + the hash of the ciphertext.
	return append(headerJSON, []byte(b.Header.CiphertextHash)...), nil
}
