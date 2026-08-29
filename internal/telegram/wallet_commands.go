package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/ethereum/go-ethereum/common"

	"github.com/richardsric7/base-bc-telegram-bot-service/internal/chain"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/storage"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/wallet"
)

func (b *Bot) cmdWalletNew(ctx context.Context, user *storage.User, chatID int64, label string) {
	label = strings.TrimSpace(label)
	if label == "" {
		b.reply(chatID, "Usage: /wallet_new <label>")
		return
	}
	w, err := b.wallet.Create(user.ID, label)
	if err != nil {
		b.reply(chatID, walletErrorMessage(err))
		return
	}
	b.reply(chatID, fmt.Sprintf("✅ Created wallet %q\nAddress: %s\n\n%s", label, w.Address, b.activationStatusLine(ctx, w.Address)))
}

func (b *Bot) cmdWalletImportStart(chatID int64, label string) {
	label = strings.TrimSpace(label)
	if label == "" {
		b.reply(chatID, "Usage: /wallet_import <label>")
		return
	}
	sess := b.sessions.start(chatID, flowWalletImport, "await_key")
	sess.Data["label"] = label
	b.reply(chatID, "Send the private key (hex, with or without 0x) for wallet \""+label+"\" now.\n\n⚠️ I will delete your message immediately after reading it. Only do this in a private chat with the bot. Send /cancel to abort.")
}

func (b *Bot) cmdWalletList(user *storage.User, chatID int64) {
	wallets, err := b.wallet.List(user.ID)
	if err != nil {
		b.reply(chatID, "Failed to list wallets: "+err.Error())
		return
	}
	if len(wallets) == 0 {
		b.reply(chatID, "You have no wallets yet. Use /wallet_new <label> or /wallet_import <label>.")
		return
	}
	var sb strings.Builder
	sb.WriteString("Your wallets:\n")
	for _, w := range wallets {
		sb.WriteString(fmt.Sprintf("• %s — %s\n", w.Label, w.Address))
	}
	b.reply(chatID, sb.String())
}

func (b *Bot) cmdWalletBalance(ctx context.Context, user *storage.User, chatID int64, label string) {
	label = strings.TrimSpace(label)
	if label == "" {
		b.reply(chatID, "Usage: /wallet_balance <label>")
		return
	}
	w, err := b.wallet.Get(user.ID, label)
	if err != nil {
		b.reply(chatID, walletErrorMessage(err))
		return
	}
	status, err := b.chain.CheckActivation(ctx, common.HexToAddress(w.Address))
	if err != nil {
		b.reply(chatID, "Failed to read balance: "+err.Error())
		return
	}
	b.reply(chatID, fmt.Sprintf(
		"%s (%s)\nBalance: %s ETH\nTransactions sent: %d\n\n%s",
		w.Label, w.Address, weiToDecimalString(status.Balance, 18), status.Nonce, formatActivation(status),
	))
}

// continueFlow handles a plain-text message while a chat is mid multi-step
// flow (wallet import, token create).
func (b *Bot) continueFlow(ctx context.Context, user *storage.User, msg *tgbotapi.Message, sess *session) {
	if strings.EqualFold(strings.TrimSpace(msg.Text), "/cancel") {
		b.sessions.clear(msg.Chat.ID)
		b.reply(msg.Chat.ID, "Cancelled.")
		return
	}

	switch sess.Flow {
	case flowWalletImport:
		b.continueWalletImport(ctx, user, msg, sess)
	case flowTokenCreate:
		b.continueTokenCreate(ctx, user, msg, sess)
	}
}

func (b *Bot) continueWalletImport(ctx context.Context, user *storage.User, msg *tgbotapi.Message, sess *session) {
	label := sess.Data["label"]
	key := strings.TrimSpace(msg.Text)

	// Best-effort: remove the message containing the private key from chat
	// history as soon as we've read it.
	if msg.MessageID != 0 {
		b.deleteMessage(msg.Chat.ID, msg.MessageID)
	}

	w, err := b.wallet.Import(user.ID, label, key)
	b.sessions.clear(msg.Chat.ID)
	if err != nil {
		b.reply(msg.Chat.ID, walletErrorMessage(err))
		return
	}
	b.reply(msg.Chat.ID, fmt.Sprintf("✅ Imported wallet %q\nAddress: %s\n\n%s", label, w.Address, b.activationStatusLine(ctx, w.Address)))
}

// activationStatusLine reads an address's on-chain nonce/balance and
// returns a one-line human summary of whether it has ever been used or
// funded. It never fails loudly — if the RPC call errors, it says so
// briefly rather than blocking the wallet creation/import flow.
func (b *Bot) activationStatusLine(ctx context.Context, address string) string {
	status, err := b.chain.CheckActivation(ctx, common.HexToAddress(address))
	if err != nil {
		return "⚠️ Couldn't check on-chain activation status: " + err.Error()
	}
	return formatActivation(status)
}

func formatActivation(status *chain.ActivationStatus) string {
	if status.Activated {
		return "✅ Activated on-chain — this address has a balance or transaction history."
	}
	return "⚠️ Not yet activated on-chain — this address has no balance and has never sent a transaction. Send it some ETH before using it to sign transactions (it needs ETH to pay gas)."
}

func walletErrorMessage(err error) string {
	if errors.Is(err, wallet.ErrLabelTaken) {
		return "That wallet label is already in use. Pick a different one."
	}
	if errors.Is(err, storage.ErrNotFound) {
		return "No wallet found with that label. Use /wallet_list to see what you have."
	}
	return "Wallet error: " + err.Error()
}
