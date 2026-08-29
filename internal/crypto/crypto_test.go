package crypto

import "testing"

func TestSealerRoundTrip(t *testing.T) {
	sealer, err := NewSealer("a-sufficiently-long-passphrase")
	if err != nil {
		t.Fatalf("NewSealer: %v", err)
	}

	plaintext := []byte("super secret private key bytes")
	ciphertext, err := sealer.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext must not equal plaintext")
	}

	decrypted, err := sealer.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestSealerWrongPassphraseFails(t *testing.T) {
	sealer, _ := NewSealer("a-sufficiently-long-passphrase")
	ciphertext, _ := sealer.Encrypt([]byte("secret"))

	other, _ := NewSealer("a-different-long-passphrase")
	if _, err := other.Decrypt(ciphertext); err == nil {
		t.Fatal("expected decrypt with wrong passphrase to fail")
	}
}

func TestNewSealerRejectsShortPassphrase(t *testing.T) {
	if _, err := NewSealer("short"); err == nil {
		t.Fatal("expected error for short passphrase")
	}
}
