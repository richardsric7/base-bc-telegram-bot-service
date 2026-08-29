// Package crypto encrypts private keys at rest with AES-256-GCM, deriving
// the key-encryption key from an operator-supplied passphrase via scrypt.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	saltLen = 16
	keyLen  = 32 // AES-256
)

// scrypt cost parameters. N must be a power of two.
const (
	scryptN = 1 << 15
	scryptR = 8
	scryptP = 1
)

// Sealer encrypts and decrypts secrets using a passphrase-derived key.
// Each call to Encrypt uses a fresh random salt and nonce, so ciphertexts
// for the same plaintext differ every time.
type Sealer struct {
	passphrase string
}

// NewSealer builds a Sealer from the operator's master passphrase.
func NewSealer(passphrase string) (*Sealer, error) {
	if len(passphrase) < 16 {
		return nil, errors.New("crypto: passphrase must be at least 16 characters")
	}
	return &Sealer{passphrase: passphrase}, nil
}

// Encrypt returns salt || nonce || ciphertext, all needed to decrypt later.
func (s *Sealer) Encrypt(plaintext []byte) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("crypto: read salt: %w", err)
	}
	key, err := deriveKey(s.passphrase, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: read nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, 0, saltLen+len(nonce)+len(ciphertext))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// Decrypt reverses Encrypt.
func (s *Sealer) Decrypt(blob []byte) ([]byte, error) {
	if len(blob) < saltLen {
		return nil, errors.New("crypto: ciphertext too short")
	}
	salt, rest := blob[:saltLen], blob[saltLen:]

	key, err := deriveKey(s.passphrase, salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}
	if len(rest) < gcm.NonceSize() {
		return nil, errors.New("crypto: ciphertext too short")
	}
	nonce, ciphertext := rest[:gcm.NonceSize()], rest[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: %w", err)
	}
	return plaintext, nil
}

func deriveKey(passphrase string, salt []byte) ([]byte, error) {
	key, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return nil, fmt.Errorf("crypto: derive key: %w", err)
	}
	return key, nil
}
