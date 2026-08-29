// Command bot is the entrypoint for the Base blockchain Telegram bot
// service: it loads configuration, opens the database, connects to Base,
// and starts long-polling Telegram for commands.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/common"

	appcrypto "github.com/richardsric7/base-bc-telegram-bot-service/internal/crypto"

	"github.com/richardsric7/base-bc-telegram-bot-service/internal/chain"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/config"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/storage"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/swap"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/telegram"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/token"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/wallet"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	store, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}

	chainClient, err := chain.Dial(cfg.BaseRPCURL, cfg.ChainID, cfg.MaxTxValueWei)
	if err != nil {
		return err
	}

	sealer, err := appcrypto.NewSealer(cfg.WalletEncryptionKey)
	if err != nil {
		return err
	}
	walletSvc := wallet.New(store, sealer)
	tokenSvc := token.New(chainClient)

	var routerAddr, quoterAddr, wethAddr common.Address
	if cfg.DEXRouterAddress != "" {
		routerAddr = common.HexToAddress(cfg.DEXRouterAddress)
	}
	if cfg.DEXQuoterAddress != "" {
		quoterAddr = common.HexToAddress(cfg.DEXQuoterAddress)
	}
	if cfg.WETHAddress != "" {
		wethAddr = common.HexToAddress(cfg.WETHAddress)
	}
	swapSvc := swap.New(chainClient, routerAddr, quoterAddr, wethAddr)

	bot, err := telegram.New(cfg, store, chainClient, walletSvc, tokenSvc, swapSvc)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("starting bot service (network=%s, chainID=%d)", cfg.ChainNetwork, cfg.ChainID)
	if err := bot.Run(ctx); err != nil && ctx.Err() == nil {
		return err
	}
	log.Println("shutting down")
	return nil
}
