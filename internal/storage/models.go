// Package storage defines the persisted data model and repositories used by
// the bot service (users, wallets, deployed tokens, and transaction log).
package storage

import "time"

// Role identifies a user's permission level.
type Role string

const (
	RoleOwner Role = "owner"
	RoleAdmin Role = "admin"
)

// User represents a Telegram user allowed to operate the bot.
type User struct {
	ID             uint  `gorm:"primarykey"`
	TelegramUserID int64 `gorm:"uniqueIndex;not null"`
	Username       string
	Role           Role `gorm:"not null;default:admin"`
	CreatedAt      time.Time
}

// Wallet is a private key managed on behalf of a User, stored encrypted at
// rest. PrivateKeyEnc holds ciphertext produced by internal/crypto; the raw
// key never touches the database or logs.
type Wallet struct {
	ID            uint   `gorm:"primarykey"`
	UserID        uint   `gorm:"not null;index"`
	Label         string `gorm:"not null"`
	Address       string `gorm:"not null;uniqueIndex"`
	PrivateKeyEnc []byte `gorm:"not null"`
	IsDefault     bool   `gorm:"not null;default:false"`
	CreatedAt     time.Time
}

// Token is an ERC-20 contract deployed through the bot.
type Token struct {
	ID              uint   `gorm:"primarykey"`
	UserID          uint   `gorm:"not null;index"`
	WalletAddress   string `gorm:"not null"`
	ChainID         int64  `gorm:"not null"`
	ContractAddress string `gorm:"not null;uniqueIndex"`
	Name            string
	Symbol          string
	Decimals        uint8
	InitialSupply   string // decimal string (base units) to avoid float precision loss
	DeployTxHash    string
	CreatedAt       time.Time
}

// TxKind classifies a logged on-chain action.
type TxKind string

const (
	TxKindDeploy        TxKind = "deploy"
	TxKindMint          TxKind = "mint"
	TxKindBurn          TxKind = "burn"
	TxKindTransferOwner TxKind = "transfer_owner"
	TxKindPause         TxKind = "pause"
	TxKindUnpause       TxKind = "unpause"
	TxKindApprove       TxKind = "approve"
	TxKindSwap          TxKind = "swap"
)

// TxStatus tracks the lifecycle of a submitted transaction.
type TxStatus string

const (
	TxStatusPending TxStatus = "pending"
	TxStatusSuccess TxStatus = "success"
	TxStatusFailed  TxStatus = "failed"
)

// Transaction is an audit log entry for every on-chain action the bot takes.
type Transaction struct {
	ID            uint     `gorm:"primarykey"`
	UserID        uint     `gorm:"not null;index"`
	WalletAddress string   `gorm:"not null;index"`
	Kind          TxKind   `gorm:"not null"`
	Status        TxStatus `gorm:"not null;default:pending"`
	TxHash        string   `gorm:"index"`
	Detail        string   // free-form human-readable summary
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// AllModels lists every model for AutoMigrate.
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&Wallet{},
		&Token{},
		&Transaction{},
	}
}
