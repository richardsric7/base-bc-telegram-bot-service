// Package telegram wires the Telegram bot interface to the chain/wallet/
// token/swap services: command routing, an allow-list auth gate, per-chat
// session state for guided multi-step flows, and inline-keyboard
// confirmation for every state-changing on-chain action.
package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/richardsric7/base-bc-telegram-bot-service/internal/chain"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/config"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/storage"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/swap"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/token"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/wallet"
)

// Bot glues the Telegram API client to the application services.
type Bot struct {
	api    *tgbotapi.BotAPI
	cfg    *config.Config
	store  *storage.Store
	chain  *chain.Client
	wallet *wallet.Service
	token  *token.Service
	swap   *swap.Service

	sessions *sessionStore
	pending  *pendingStore
}

// New builds a Bot ready to Run.
func New(cfg *config.Config, store *storage.Store, chainClient *chain.Client, walletSvc *wallet.Service, tokenSvc *token.Service, swapSvc *swap.Service) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("telegram: init bot api: %w", err)
	}
	return &Bot{
		api:      api,
		cfg:      cfg,
		store:    store,
		chain:    chainClient,
		wallet:   walletSvc,
		token:    tokenSvc,
		swap:     swapSvc,
		sessions: newSessionStore(),
		pending:  newPendingStore(),
	}, nil
}

// Run starts long-polling for updates and blocks until ctx is done.
func (b *Bot) Run(ctx context.Context) error {
	log.Printf("telegram: authorized as @%s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return ctx.Err()
		case update := <-updates:
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("telegram: recovered panic handling update: %v", r)
		}
	}()

	switch {
	case update.CallbackQuery != nil:
		b.handleCallback(ctx, update.CallbackQuery)
	case update.Message != nil:
		b.handleMessage(ctx, update.Message)
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	if msg.From == nil {
		return
	}
	if !b.cfg.IsAllowed(msg.From.ID) {
		b.reply(msg.Chat.ID, "Unauthorized. This bot is restricted to a configured allow-list of Telegram user IDs.")
		return
	}

	user, err := b.store.GetOrCreateUser(msg.From.ID, msg.From.UserName, storage.RoleOwner)
	if err != nil {
		log.Printf("telegram: get/create user: %v", err)
		b.reply(msg.Chat.ID, "Internal error loading your account. Please try again.")
		return
	}

	// If this chat is mid-flow (e.g. awaiting a private key or a token
	// field) and the message isn't itself a new command, feed it to the
	// active flow instead of the command router.
	if sess, ok := b.sessions.get(msg.Chat.ID); ok && sess.Flow != flowNone && !msg.IsCommand() {
		b.continueFlow(ctx, user, msg, sess)
		return
	}

	if !msg.IsCommand() {
		return
	}

	cmd := msg.Command()
	args := strings.TrimSpace(msg.CommandArguments())

	switch cmd {
	case "start", "help":
		b.cmdHelp(msg.Chat.ID)
	case "network":
		b.cmdNetwork(ctx, msg.Chat.ID)
	case "tx":
		b.cmdTx(msg.Chat.ID, args)

	case "wallet_new":
		b.cmdWalletNew(user, msg.Chat.ID, args)
	case "wallet_import":
		b.cmdWalletImportStart(msg.Chat.ID, args)
	case "wallet_list":
		b.cmdWalletList(user, msg.Chat.ID)
	case "wallet_balance":
		b.cmdWalletBalance(ctx, user, msg.Chat.ID, args)

	case "token_create":
		b.cmdTokenCreateStart(user, msg.Chat.ID)
	case "token_list":
		b.cmdTokenList(user, msg.Chat.ID)
	case "token_info":
		b.cmdTokenInfo(ctx, msg.Chat.ID, args)
	case "token_mint":
		b.cmdTokenMint(ctx, user, msg.Chat.ID, msg.From.ID, args)
	case "token_burn":
		b.cmdTokenBurn(ctx, user, msg.Chat.ID, msg.From.ID, args)
	case "token_transfer_owner":
		b.cmdTokenTransferOwner(ctx, user, msg.Chat.ID, msg.From.ID, args)
	case "token_pause":
		b.cmdTokenPause(ctx, user, msg.Chat.ID, msg.From.ID, args, true)
	case "token_unpause":
		b.cmdTokenPause(ctx, user, msg.Chat.ID, msg.From.ID, args, false)

	case "swap_quote":
		b.cmdSwapQuote(ctx, msg.Chat.ID, args)
	case "swap_execute":
		b.cmdSwapExecute(ctx, user, msg.Chat.ID, msg.From.ID, args)

	default:
		b.reply(msg.Chat.ID, "Unknown command. Send /help to see everything I support.")
	}
}

func (b *Bot) handleCallback(ctx context.Context, cq *tgbotapi.CallbackQuery) {
	if cq.From == nil || !b.cfg.IsAllowed(cq.From.ID) {
		b.answerCallback(cq.ID, "Unauthorized")
		return
	}

	data := cq.Data
	action, token, found := strings.Cut(data, ":")
	if !found {
		b.answerCallback(cq.ID, "")
		return
	}

	pending, ok := b.pending.take(token)
	if !ok {
		b.answerCallback(cq.ID, "This confirmation has expired or was already used.")
		return
	}
	if pending.UserTelegramID != cq.From.ID {
		b.answerCallback(cq.ID, "This confirmation isn't yours.")
		return
	}

	switch action {
	case "cancel":
		b.answerCallback(cq.ID, "Cancelled.")
		b.editMessage(cq.Message.Chat.ID, cq.Message.MessageID, pending.Description+"\n\n❌ Cancelled.")
	case "confirm":
		b.answerCallback(cq.ID, "Submitting…")
		b.editMessage(cq.Message.Chat.ID, cq.Message.MessageID, pending.Description+"\n\n⏳ Submitting to the chain, please wait…")
		result, err := pending.Execute()
		if err != nil {
			b.editMessage(cq.Message.Chat.ID, cq.Message.MessageID, pending.Description+"\n\n❌ Failed: "+err.Error())
			return
		}
		b.editMessage(cq.Message.Chat.ID, cq.Message.MessageID, pending.Description+"\n\n✅ "+result)
	default:
		b.answerCallback(cq.ID, "")
	}
	_ = ctx
}

// confirm sends msg with Confirm/Cancel buttons and registers execute to run
// only if the same Telegram user taps Confirm.
func (b *Bot) confirm(chatID, telegramUserID int64, description string, execute func() (string, error)) {
	pending := &pendingAction{
		UserTelegramID: telegramUserID,
		ChatID:         chatID,
		Description:    description,
		Execute:        execute,
	}
	token, err := b.pending.add(pending)
	if err != nil {
		b.reply(chatID, "Internal error preparing confirmation.")
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Confirm", "confirm:"+token),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "cancel:"+token),
		),
	)
	m := tgbotapi.NewMessage(chatID, description)
	m.ReplyMarkup = keyboard
	if _, err := b.api.Send(m); err != nil {
		log.Printf("telegram: send confirm: %v", err)
	}
}

func (b *Bot) reply(chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(m); err != nil {
		log.Printf("telegram: send message: %v", err)
	}
}

func (b *Bot) editMessage(chatID int64, messageID int, text string) {
	m := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if _, err := b.api.Send(m); err != nil {
		log.Printf("telegram: edit message: %v", err)
	}
}

func (b *Bot) answerCallback(id, text string) {
	cb := tgbotapi.NewCallback(id, text)
	if _, err := b.api.Request(cb); err != nil {
		log.Printf("telegram: answer callback: %v", err)
	}
}

func (b *Bot) deleteMessage(chatID int64, messageID int) {
	del := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := b.api.Request(del); err != nil {
		log.Printf("telegram: delete message: %v", err)
	}
}

func (b *Bot) cmdHelp(chatID int64) {
	b.reply(chatID, strings.TrimSpace(`
*Base Telegram Bot* — create and manage ERC-20 tokens and swap assets on Base, from wallets you control.

*Wallets*
/wallet_new <label> — generate a new wallet
/wallet_import <label> — import a wallet by private key (bot will ask for it privately)
/wallet_list — list your wallets
/wallet_balance <label> — ETH balance of a wallet

*Tokens*
/token_create — guided flow to deploy a new ERC-20 token
/token_list — tokens you've deployed
/token_info <address> — read a token's on-chain info
/token_mint <address> <to> <amount>
/token_burn <walletLabel> <address> <amount>
/token_transfer_owner <address> <newOwner>
/token_pause <address>
/token_unpause <address>

*Swaps*
/swap_quote <fromToken|ETH> <toToken|ETH> <amount>
/swap_execute <wallet> <fromToken|ETH> <toToken|ETH> <amount> <slippageBps>

*Utility*
/tx <hash> — look up a transaction
/network — current chain + gas price

All state-changing actions ask for confirmation before anything is sent on-chain.
`))
}

func (b *Bot) cmdNetwork(ctx context.Context, chatID int64) {
	gasPrice, err := b.chain.SuggestGasPrice(ctx)
	if err != nil {
		b.reply(chatID, "Failed to read network status: "+err.Error())
		return
	}
	b.reply(chatID, fmt.Sprintf(
		"Network: %s\nChain ID: %s\nRPC: %s\nSuggested gas price: %s gwei",
		b.cfg.ChainNetwork, b.chain.ChainID().String(), b.cfg.BaseRPCURL,
		weiToDecimalString(gasPrice, 9),
	))
}

func (b *Bot) cmdTx(chatID int64, args string) {
	hash := strings.TrimSpace(args)
	if hash == "" {
		b.reply(chatID, "Usage: /tx <hash>")
		return
	}
	b.reply(chatID, "🔗 "+b.chain.TxURL(hash))
}
