// Package wallet manages ECDSA key pairs on behalf of bot users: generation,
// import, encrypted-at-rest storage, and balance lookups. Raw private keys
// only ever exist in memory for the duration of a signing operation.
package wallet

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	appcrypto "github.com/richardsric7/base-bc-telegram-bot-service/internal/crypto"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/storage"
)

// Service provides wallet management operations.
type Service struct {
	store  *storage.Store
	sealer *appcrypto.Sealer
}

// New builds a wallet Service.
func New(store *storage.Store, sealer *appcrypto.Sealer) *Service {
	return &Service{store: store, sealer: sealer}
}

// ErrLabelTaken is returned when a wallet label is already used by the user.
var ErrLabelTaken = errors.New("wallet: label already in use")

// Create generates a brand-new secp256k1 key pair, encrypts the private key,
// and stores it as a wallet owned by userID.
func (s *Service) Create(userID uint, label string) (*storage.Wallet, error) {
	if _, err := s.store.GetWalletByLabel(userID, label); err == nil {
		return nil, ErrLabelTaken
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("wallet: generate key: %w", err)
	}
	return s.persist(userID, label, key)
}

// Import stores an operator-supplied private key (hex, with or without 0x
// prefix) as a new wallet.
func (s *Service) Import(userID uint, label, privateKeyHex string) (*storage.Wallet, error) {
	if _, err := s.store.GetWalletByLabel(userID, label); err == nil {
		return nil, ErrLabelTaken
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}

	privateKeyHex = strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("wallet: invalid private key: %w", err)
	}
	return s.persist(userID, label, key)
}

func (s *Service) persist(userID uint, label string, key *ecdsa.PrivateKey) (*storage.Wallet, error) {
	address := crypto.PubkeyToAddress(key.PublicKey).Hex()
	keyBytes := crypto.FromECDSA(key)
	enc, err := s.sealer.Encrypt(keyBytes)
	// Zero the plaintext key material as soon as we're done with it.
	for i := range keyBytes {
		keyBytes[i] = 0
	}
	if err != nil {
		return nil, fmt.Errorf("wallet: encrypt key: %w", err)
	}

	w := &storage.Wallet{
		UserID:        userID,
		Label:         label,
		Address:       address,
		PrivateKeyEnc: enc,
	}
	if err := s.store.CreateWallet(w); err != nil {
		return nil, fmt.Errorf("wallet: persist: %w", err)
	}
	return w, nil
}

// List returns all wallets owned by userID (addresses only; never decrypts
// keys).
func (s *Service) List(userID uint) ([]storage.Wallet, error) {
	return s.store.ListWallets(userID)
}

// Get resolves a wallet by label for the given user.
func (s *Service) Get(userID uint, label string) (*storage.Wallet, error) {
	return s.store.GetWalletByLabel(userID, label)
}

// PrivateKey decrypts and returns the ECDSA private key for a stored wallet.
// Callers must use it immediately for signing and let it go out of scope;
// it is never logged or persisted in plaintext.
func (s *Service) PrivateKey(w *storage.Wallet) (*ecdsa.PrivateKey, error) {
	plain, err := s.sealer.Decrypt(w.PrivateKeyEnc)
	if err != nil {
		return nil, fmt.Errorf("wallet: decrypt key: %w", err)
	}
	defer func() {
		for i := range plain {
			plain[i] = 0
		}
	}()
	key, err := crypto.ToECDSA(plain)
	if err != nil {
		return nil, fmt.Errorf("wallet: parse key: %w", err)
	}
	return key, nil
}
