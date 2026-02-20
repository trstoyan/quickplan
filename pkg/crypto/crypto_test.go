package crypto

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

func TestEd25519(t *testing.T) {
	pub, priv, err := GenerateEd25519()
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("hello world")
	sig := ed25519.Sign(priv, msg)

	if !ed25519.Verify(pub, msg, sig) {
		t.Error("signature verification failed")
	}

	msg[0] = 'H'
	if ed25519.Verify(pub, msg, sig) {
		t.Error("signature verified tampered message")
	}
}

func TestAESGCM(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("secret data")
	aad := []byte("metadata")

	nonce, ciphertext, err := Encrypt(key, plaintext, aad)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := Decrypt(key, nonce, ciphertext, aad)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("decrypted data does not match original")
	}

	// Test wrong AAD
	_, err = Decrypt(key, nonce, ciphertext, []byte("wrong metadata"))
	if err == nil {
		t.Error("decryption should fail with wrong AAD")
	}
}

func TestKeyWrap(t *testing.T) {
	senderPriv, _ := GenerateX25519()
	senderPub := senderPriv.PublicKey()

	recipientPriv, _ := GenerateX25519()
	recipientPub := recipientPriv.PublicKey()

	projectKey := []byte("this-is-a-32-byte-long-proj-key!")

	nonce, wrapped, err := WrapKeyX25519(projectKey, recipientPub, senderPriv)
	if err != nil {
		t.Fatal(err)
	}

	unwrapped, err := UnwrapKeyX25519(nonce, wrapped, recipientPriv, senderPub)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(projectKey, unwrapped) {
		t.Error("unwrapped key does not match original")
	}
}
