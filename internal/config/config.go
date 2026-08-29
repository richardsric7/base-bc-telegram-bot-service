// Package config loads and validates service configuration from the
// environment (optionally via a .env file).
package config

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the bot service.
type Config struct {
	TelegramBotToken   string
	AllowedTelegramIDs map[int64]struct{}

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

// Load reads configuration from the process environment, first merging in
// values from a .env file in the working directory if one is present.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
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

	ids, err := parseAllowedIDs(os.Getenv("ALLOWED_TELEGRAM_IDS"))
	if err != nil {
		return nil, err
	}
	cfg.AllowedTelegramIDs = ids

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
	if len(c.AllowedTelegramIDs) == 0 {
		return fmt.Errorf("config: ALLOWED_TELEGRAM_IDS must list at least one Telegram user ID")
	}
	if len(c.WalletEncryptionKey) < 16 {
		return fmt.Errorf("config: WALLET_ENCRYPTION_KEY is required and must be at least 16 characters")
	}
	if c.BaseRPCURL == "" {
		return fmt.Errorf("config: BASE_RPC_URL is required")
	}
	return nil
}

// IsAllowed reports whether the given Telegram user ID is authorized to use
// the bot.
func (c *Config) IsAllowed(telegramUserID int64) bool {
	_, ok := c.AllowedTelegramIDs[telegramUserID]
	return ok
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseAllowedIDs(raw string) (map[int64]struct{}, error) {
	ids := make(map[int64]struct{})
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ids, nil
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("config: invalid ALLOWED_TELEGRAM_IDS entry %q: %w", part, err)
		}
		ids[id] = struct{}{}
	}
	return ids, nil
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
