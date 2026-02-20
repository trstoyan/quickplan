package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"
)

// GenerateEd25519 returns a new Ed25519 key pair.
func GenerateEd25519() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// GenerateX25519 returns a new X25519 key pair.
func GenerateX25519() (*ecdh.PrivateKey, error) {
	return ecdh.X25519().GenerateKey(rand.Reader)
}

// Encrypt encrypts plaintext using AES-256-GCM with associated data (AAD).
func Encrypt(key32 []byte, plaintext []byte, aad []byte) (nonce []byte, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key32)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, aad)
	return nonce, ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with associated data (AAD).
func Decrypt(key32 []byte, nonce []byte, ciphertext []byte, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key32)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, aad)
}

// DeriveSharedKey derives a 32-byte symmetric key from an X25519 shared secret.
func DeriveSharedKey(priv *ecdh.PrivateKey, pub *ecdh.PublicKey, salt []byte) ([]byte, error) {
	secret, err := priv.ECDH(pub)
	if err != nil {
		return nil, err
	}

	hkdfReader := hkdf.New(sha256.New, secret, salt, []byte("quickplan-keystore-v1"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, err
	}

	return key, nil
}

// WrapKeyX25519 wraps a 32-byte project key using X25519 and AES-GCM.
func WrapKeyX25519(projectKey []byte, recipientPub *ecdh.PublicKey, senderPriv *ecdh.PrivateKey) (nonce []byte, wrappedKey []byte, err error) {
	kek, err := DeriveSharedKey(senderPriv, recipientPub, nil)
	if err != nil {
		return nil, nil, err
	}
	return Encrypt(kek, projectKey, nil)
}

// UnwrapKeyX25519 unwraps a 32-byte project key.
func UnwrapKeyX25519(nonce []byte, wrappedKey []byte, recipientPriv *ecdh.PrivateKey, senderPub *ecdh.PublicKey) ([]byte, error) {
	kek, err := DeriveSharedKey(recipientPriv, senderPub, nil)
	if err != nil {
		return nil, err
	}
	return Decrypt(kek, nonce, wrappedKey, nil)
}
