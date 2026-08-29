package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Store wraps a gorm DB handle and provides typed repository methods.
type Store struct {
	db *gorm.DB
}

// Open opens (creating if needed) the SQLite database at path and runs
// AutoMigrate for all models.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("storage: create db directory: %w", err)
		}
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: open db: %w", err)
	}
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return nil, fmt.Errorf("storage: migrate: %w", err)
	}
	return &Store{db: db}, nil
}

// ErrNotFound is returned when a lookup finds no matching row.
var ErrNotFound = gorm.ErrRecordNotFound

// GetOrCreateUser fetches the user for a Telegram ID, creating one (as the
// given role) if it doesn't exist yet.
func (s *Store) GetOrCreateUser(telegramUserID int64, username string, defaultRole Role) (*User, error) {
	var u User
	err := s.db.Where("telegram_user_id = ?", telegramUserID).First(&u).Error
	if err == nil {
		if username != "" && u.Username != username {
			u.Username = username
			s.db.Save(&u)
		}
		return &u, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	u = User{TelegramUserID: telegramUserID, Username: username, Role: defaultRole}
	if err := s.db.Create(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateWallet persists a new wallet row.
func (s *Store) CreateWallet(w *Wallet) error {
	return s.db.Create(w).Error
}

// ListWallets returns all wallets owned by a user.
func (s *Store) ListWallets(userID uint) ([]Wallet, error) {
	var wallets []Wallet
	err := s.db.Where("user_id = ?", userID).Order("created_at asc").Find(&wallets).Error
	return wallets, err
}

// GetWalletByLabel finds a user's wallet by its label.
func (s *Store) GetWalletByLabel(userID uint, label string) (*Wallet, error) {
	var w Wallet
	err := s.db.Where("user_id = ? AND label = ?", userID, label).First(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// GetWalletByAddress finds a user's wallet by its on-chain address.
func (s *Store) GetWalletByAddress(userID uint, address string) (*Wallet, error) {
	var w Wallet
	err := s.db.Where("user_id = ? AND address = ?", userID, address).First(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// CreateToken persists a new deployed-token row.
func (s *Store) CreateToken(t *Token) error {
	return s.db.Create(t).Error
}

// ListTokens returns all tokens deployed by a user.
func (s *Store) ListTokens(userID uint) ([]Token, error) {
	var tokens []Token
	err := s.db.Where("user_id = ?", userID).Order("created_at asc").Find(&tokens).Error
	return tokens, err
}

// GetTokenByAddress finds a token this user deployed by contract address.
func (s *Store) GetTokenByAddress(userID uint, address string) (*Token, error) {
	var t Token
	err := s.db.Where("user_id = ? AND contract_address = ?", userID, address).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// LogTransaction persists a transaction audit row.
func (s *Store) LogTransaction(t *Transaction) error {
	return s.db.Create(t).Error
}

// UpdateTransactionStatus updates the status/hash of a previously logged
// transaction.
func (s *Store) UpdateTransactionStatus(id uint, status TxStatus, txHash string) error {
	return s.db.Model(&Transaction{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":  status,
		"tx_hash": txHash,
	}).Error
}
