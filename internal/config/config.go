// Package config loads and validates service configuration from the
// environment (optionally via a .env file).
package config

import (
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the bot service.
type Config struct {
	TelegramBotToken string

	// AdminSetupCode is a 6-digit shared secret. The first Telegram user to
	// send this code to the bot is registered as its admin; nobody else can
	// claim the admin role this way once an admin exists. This replaces a
	// static allow-list of Telegram IDs with a one-time pairing code, so no
	// IDs need to be known or configured up front.
	AdminSetupCode string

	BaseRPCURL   string
	ChainNetwork string // "sepolia" or "mainnet"
	ChainID      int64

	WalletEncryptionKey string

	DEXRouterAddress string
	DEXQuoterAddress string
	WETHAddress      string

	DatabasePath string

	// MaxTxValueWei is the per-transaction native-value safety cap in wei.
	// A zero value means no cap.
	MaxTxValueWei *big.Int
}

const (
	baseMainnetChainID = 8453
	baseSepoliaChainID = 84532
)

var sixDigitCode = regexp.MustCompile(`^\d{6}$`)

// Load reads configuration from the process environment, first merging in
// values from a .env file in the working directory if one is present.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		AdminSetupCode:      os.Getenv("ADMIN_SETUP_CODE"),
		BaseRPCURL:          getEnvDefault("BASE_RPC_URL", "https://sepolia.base.org"),
		ChainNetwork:        strings.ToLower(getEnvDefault("CHAIN_NETWORK", "sepolia")),
		WalletEncryptionKey: os.Getenv("WALLET_ENCRYPTION_KEY"),
		DEXRouterAddress:    os.Getenv("DEX_ROUTER_ADDRESS"),
		DEXQuoterAddress:    os.Getenv("DEX_QUOTER_ADDRESS"),
		WETHAddress:         os.Getenv("WETH_ADDRESS"),
		DatabasePath:        getEnvDefault("DATABASE_PATH", "./data/bot.db"),
	}

	switch cfg.ChainNetwork {
	case "mainnet":
		cfg.ChainID = baseMainnetChainID
	case "sepolia":
		cfg.ChainID = baseSepoliaChainID
	default:
		return nil, fmt.Errorf("config: invalid CHAIN_NETWORK %q (want \"mainnet\" or \"sepolia\")", cfg.ChainNetwork)
	}

	maxTx, err := parseMaxTxValue(os.Getenv("MAX_TX_VALUE_ETH"))
	if err != nil {
		return nil, err
	}
	cfg.MaxTxValueWei = maxTx

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.TelegramBotToken == "" {
		return fmt.Errorf("config: TELEGRAM_BOT_TOKEN is required")
	}
	if !sixDigitCode.MatchString(c.AdminSetupCode) {
		return fmt.Errorf("config: ADMIN_SETUP_CODE is required and must be exactly 6 digits")
	}
	if len(c.WalletEncryptionKey) < 16 {
		return fmt.Errorf("config: WALLET_ENCRYPTION_KEY is required and must be at least 16 characters")
	}
	if c.BaseRPCURL == "" {
		return fmt.Errorf("config: BASE_RPC_URL is required")
	}
	return nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseMaxTxValue(raw string) (*big.Int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return big.NewInt(0), nil
	}
	f, ok := new(big.Float).SetString(raw)
	if !ok {
		return nil, fmt.Errorf("config: invalid MAX_TX_VALUE_ETH %q", raw)
	}
	wei := new(big.Float).Mul(f, big.NewFloat(1e18))
	weiInt, _ := wei.Int(nil)
	return weiInt, nil
}
