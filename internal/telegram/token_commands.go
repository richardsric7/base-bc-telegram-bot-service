package telegram

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/ethereum/go-ethereum/common"

	"github.com/richardsric7/base-bc-telegram-bot-service/internal/storage"
)

// --- /token_create guided flow -------------------------------------------

func (b *Bot) cmdTokenCreateStart(user *storage.User, chatID int64) {
	wallets, err := b.wallet.List(user.ID)
	if err != nil || len(wallets) == 0 {
		b.reply(chatID, "You need at least one wallet first. Use /wallet_new <label> or /wallet_import <label>.")
		return
	}
	sess := b.sessions.start(chatID, flowTokenCreate, "await_name")
	b.reply(chatID, "Let's create a new ERC-20 token.\n\nWhat's the token *name*? (e.g. \"My Token\")")
	_ = sess
}

func (b *Bot) continueTokenCreate(ctx context.Context, user *storage.User, msg *tgbotapi.Message, sess *session) {
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	switch sess.Step {
	case "await_name":
		if text == "" {
			b.reply(chatID, "Name can't be empty. What's the token name?")
			return
		}
		sess.Data["name"] = text
		sess.Step = "await_symbol"
		b.reply(chatID, "What's the token *symbol*? (e.g. \"MTK\")")

	case "await_symbol":
		if text == "" {
			b.reply(chatID, "Symbol can't be empty. What's the token symbol?")
			return
		}
		sess.Data["symbol"] = strings.ToUpper(text)
		sess.Step = "await_decimals"
		b.reply(chatID, "How many *decimals*? Send a number 0-18, or \"skip\" for the default (18).")

	case "await_decimals":
		decimals := 18
		if !strings.EqualFold(text, "skip") {
			d, err := strconv.Atoi(text)
			if err != nil || d < 0 || d > 18 {
				b.reply(chatID, "Please send a whole number between 0 and 18, or \"skip\".")
				return
			}
			decimals = d
		}
		sess.Data["decimals"] = strconv.Itoa(decimals)
		sess.Step = "await_supply"
		b.reply(chatID, fmt.Sprintf("What's the *initial supply* (in whole tokens, decimals=%d)? Send 0 for none.", decimals))

	case "await_supply":
		decimals, _ := strconv.Atoi(sess.Data["decimals"])
		supplyWei, err := decimalStringToWei(text, uint8(decimals))
		if err != nil {
			b.reply(chatID, "Invalid amount: "+err.Error()+". Try again.")
			return
		}
		sess.Data["supply_wei"] = supplyWei.String()
		sess.Step = "await_wallet"

		wallets, err := b.wallet.List(user.ID)
		if err != nil || len(wallets) == 0 {
			b.sessions.clear(chatID)
			b.reply(chatID, "No wallets available anymore. Use /wallet_new first, then /token_create again.")
			return
		}
		var sb strings.Builder
		sb.WriteString("Which *wallet* should deploy and own this token? Send its label.\n\n")
		for _, w := range wallets {
			sb.WriteString(fmt.Sprintf("• %s (%s)\n", w.Label, shortAddr(w.Address)))
		}
		b.reply(chatID, sb.String())

	case "await_wallet":
		w, err := b.wallet.Get(user.ID, text)
		if err != nil {
			b.reply(chatID, "No wallet with that label. Try again, or /cancel.")
			return
		}
		sess.Data["wallet_label"] = w.Label
		sess.Data["wallet_address"] = w.Address
		b.sessions.clear(chatID)

		decimals, _ := strconv.Atoi(sess.Data["decimals"])
		supplyWei, _ := new(big.Int).SetString(sess.Data["supply_wei"], 10)
		name, symbol := sess.Data["name"], sess.Data["symbol"]

		description := fmt.Sprintf(
			"Deploy ERC-20 token:\n• Name: %s\n• Symbol: %s\n• Decimals: %d\n• Initial supply: %s\n• Deployer/owner wallet: %s (%s)\n• Network: %s",
			name, symbol, decimals, weiToDecimalString(supplyWei, uint8(decimals)), w.Label, w.Address, b.cfg.ChainNetwork,
		)

		b.confirm(chatID, msg.From.ID, description, func() (string, error) {
			key, err := b.wallet.PrivateKey(w)
			if err != nil {
				return "", err
			}
			result, err := b.token.Deploy(ctx, key, name, symbol, uint8(decimals), supplyWei)
			if err != nil {
				return "", err
			}
			_ = b.store.CreateToken(&storage.Token{
				UserID:          user.ID,
				WalletAddress:   w.Address,
				ChainID:         b.chain.ChainID().Int64(),
				ContractAddress: result.Address.Hex(),
				Name:            name,
				Symbol:          symbol,
				Decimals:        uint8(decimals),
				InitialSupply:   supplyWei.String(),
				DeployTxHash:    result.Tx.Hash().Hex(),
			})
			_ = b.store.LogTransaction(&storage.Transaction{
				UserID:        user.ID,
				WalletAddress: w.Address,
				Kind:          storage.TxKindDeploy,
				Status:        storage.TxStatusSuccess,
				TxHash:        result.Tx.Hash().Hex(),
				Detail:        fmt.Sprintf("Deployed %s (%s) at %s", name, symbol, result.Address.Hex()),
			})
			return fmt.Sprintf("Deployed at %s\n%s", result.Address.Hex(), b.chain.TxURL(result.Tx.Hash().Hex())), nil
		})
	}
}

// --- read-only / simple token commands ------------------------------------

func (b *Bot) cmdTokenList(user *storage.User, chatID int64) {
	tokens, err := b.store.ListTokens(user.ID)
	if err != nil {
		b.reply(chatID, "Failed to list tokens: "+err.Error())
		return
	}
	if len(tokens) == 0 {
		b.reply(chatID, "You haven't deployed any tokens yet. Use /token_create.")
		return
	}
	var sb strings.Builder
	sb.WriteString("Your tokens:\n")
	for _, t := range tokens {
		sb.WriteString(fmt.Sprintf("• %s (%s) — %s\n", t.Name, t.Symbol, t.ContractAddress))
	}
	b.reply(chatID, sb.String())
}

func (b *Bot) cmdTokenInfo(ctx context.Context, chatID int64, args string) {
	addr := strings.TrimSpace(args)
	if !common.IsHexAddress(addr) {
		b.reply(chatID, "Usage: /token_info <address>")
		return
	}
	info, err := b.token.Info(ctx, common.HexToAddress(addr))
	if err != nil {
		b.reply(chatID, "Failed to read token: "+err.Error())
		return
	}
	b.reply(chatID, fmt.Sprintf(
		"%s (%s)\nAddress: %s\nDecimals: %d\nTotal supply: %s\nOwner: %s\nPaused: %v",
		info.Name, info.Symbol, info.Address.Hex(), info.Decimals,
		weiToDecimalString(info.TotalSupply, info.Decimals), info.Owner.Hex(), info.Paused,
	))
}

// --- state-changing token commands (all confirm before broadcasting) ------

func (b *Bot) cmdTokenMint(ctx context.Context, user *storage.User, chatID, telegramUserID int64, args string) {
	parts := strings.Fields(args)
	if len(parts) != 3 {
		b.reply(chatID, "Usage: /token_mint <tokenAddress> <toAddress> <amount>")
		return
	}
	tokenAddr, toAddr, amountStr := parts[0], parts[1], parts[2]
	if !common.IsHexAddress(tokenAddr) || !common.IsHexAddress(toAddr) {
		b.reply(chatID, "Invalid address.")
		return
	}

	info, err := b.token.Info(ctx, common.HexToAddress(tokenAddr))
	if err != nil {
		b.reply(chatID, "Failed to read token: "+err.Error())
		return
	}
	amount, err := decimalStringToWei(amountStr, info.Decimals)
	if err != nil {
		b.reply(chatID, "Invalid amount: "+err.Error())
		return
	}

	w, err := b.walletForOwner(user, info.Owner)
	if err != nil {
		b.reply(chatID, err.Error())
		return
	}

	description := fmt.Sprintf("Mint %s %s to %s\nToken: %s\nSigner: %s", amountStr, info.Symbol, toAddr, tokenAddr, w.Label)
	b.confirm(chatID, telegramUserID, description, func() (string, error) {
		key, err := b.wallet.PrivateKey(w)
		if err != nil {
			return "", err
		}
		receipt, err := b.token.Mint(ctx, key, common.HexToAddress(tokenAddr), common.HexToAddress(toAddr), amount)
		if err != nil {
			return "", err
		}
		b.logTx(user, w.Address, storage.TxKindMint, receipt.TxHash.Hex(), description)
		return b.chain.TxURL(receipt.TxHash.Hex()), nil
	})
}

func (b *Bot) cmdTokenBurn(ctx context.Context, user *storage.User, chatID, telegramUserID int64, args string) {
	parts := strings.Fields(args)
	if len(parts) != 3 {
		b.reply(chatID, "Usage: /token_burn <walletLabel> <tokenAddress> <amount>")
		return
	}
	walletLabel, tokenAddr, amountStr := parts[0], parts[1], parts[2]
	if !common.IsHexAddress(tokenAddr) {
		b.reply(chatID, "Invalid token address.")
		return
	}
	info, err := b.token.Info(ctx, common.HexToAddress(tokenAddr))
	if err != nil {
		b.reply(chatID, "Failed to read token: "+err.Error())
		return
	}
	amount, err := decimalStringToWei(amountStr, info.Decimals)
	if err != nil {
		b.reply(chatID, "Invalid amount: "+err.Error())
		return
	}

	w, err := b.wallet.Get(user.ID, walletLabel)
	if err != nil {
		b.reply(chatID, walletErrorMessage(err))
		return
	}

	description := fmt.Sprintf("Burn %s %s\nToken: %s\nFrom wallet: %s (%s)", amountStr, info.Symbol, tokenAddr, w.Label, w.Address)
	b.confirm(chatID, telegramUserID, description, func() (string, error) {
		key, err := b.wallet.PrivateKey(w)
		if err != nil {
			return "", err
		}
		receipt, err := b.token.Burn(ctx, key, common.HexToAddress(tokenAddr), amount)
		if err != nil {
			return "", err
		}
		b.logTx(user, w.Address, storage.TxKindBurn, receipt.TxHash.Hex(), description)
		return b.chain.TxURL(receipt.TxHash.Hex()), nil
	})
}

func (b *Bot) cmdTokenTransferOwner(ctx context.Context, user *storage.User, chatID, telegramUserID int64, args string) {
	parts := strings.Fields(args)
	if len(parts) != 2 {
		b.reply(chatID, "Usage: /token_transfer_owner <tokenAddress> <newOwnerAddress>")
		return
	}
	tokenAddr, newOwner := parts[0], parts[1]
	if !common.IsHexAddress(tokenAddr) || !common.IsHexAddress(newOwner) {
		b.reply(chatID, "Invalid address.")
		return
	}
	info, err := b.token.Info(ctx, common.HexToAddress(tokenAddr))
	if err != nil {
		b.reply(chatID, "Failed to read token: "+err.Error())
		return
	}
	w, err := b.walletForOwner(user, info.Owner)
	if err != nil {
		b.reply(chatID, err.Error())
		return
	}

	description := fmt.Sprintf("⚠️ Transfer ownership of %s (%s)\nTo: %s\nThis cannot be undone unless the new owner transfers it back.", tokenAddr, info.Symbol, newOwner)
	b.confirm(chatID, telegramUserID, description, func() (string, error) {
		key, err := b.wallet.PrivateKey(w)
		if err != nil {
			return "", err
		}
		receipt, err := b.token.TransferOwnership(ctx, key, common.HexToAddress(tokenAddr), common.HexToAddress(newOwner))
		if err != nil {
			return "", err
		}
		b.logTx(user, w.Address, storage.TxKindTransferOwner, receipt.TxHash.Hex(), description)
		return b.chain.TxURL(receipt.TxHash.Hex()), nil
	})
}

func (b *Bot) cmdTokenPause(ctx context.Context, user *storage.User, chatID, telegramUserID int64, args string, pause bool) {
	tokenAddr := strings.TrimSpace(args)
	if !common.IsHexAddress(tokenAddr) {
		action := "token_pause"
		if !pause {
			action = "token_unpause"
		}
		b.reply(chatID, fmt.Sprintf("Usage: /%s <tokenAddress>", action))
		return
	}
	info, err := b.token.Info(ctx, common.HexToAddress(tokenAddr))
	if err != nil {
		b.reply(chatID, "Failed to read token: "+err.Error())
		return
	}
	w, err := b.walletForOwner(user, info.Owner)
	if err != nil {
		b.reply(chatID, err.Error())
		return
	}

	verb, kind := "Pause", storage.TxKindPause
	if !pause {
		verb, kind = "Unpause", storage.TxKindUnpause
	}
	description := fmt.Sprintf("%s token %s (%s)", verb, tokenAddr, info.Symbol)
	b.confirm(chatID, telegramUserID, description, func() (string, error) {
		key, err := b.wallet.PrivateKey(w)
		if err != nil {
			return "", err
		}
		var hash string
		if pause {
			r, err := b.token.Pause(ctx, key, common.HexToAddress(tokenAddr))
			if err != nil {
				return "", err
			}
			hash = r.TxHash.Hex()
		} else {
			r, err := b.token.Unpause(ctx, key, common.HexToAddress(tokenAddr))
			if err != nil {
				return "", err
			}
			hash = r.TxHash.Hex()
		}
		b.logTx(user, w.Address, kind, hash, description)
		return b.chain.TxURL(hash), nil
	})
}

// walletForOwner finds a wallet belonging to user whose address matches
// owner (case-insensitive), i.e. the wallet that can sign owner-only calls.
func (b *Bot) walletForOwner(user *storage.User, owner common.Address) (*storage.Wallet, error) {
	wallets, err := b.wallet.List(user.ID)
	if err != nil {
		return nil, err
	}
	for i := range wallets {
		if strings.EqualFold(wallets[i].Address, owner.Hex()) {
			return &wallets[i], nil
		}
	}
	return nil, fmt.Errorf("none of your wallets match the token owner (%s); use /wallet_import to add it", owner.Hex())
}

func (b *Bot) logTx(user *storage.User, walletAddress string, kind storage.TxKind, txHash, detail string) {
	_ = b.store.LogTransaction(&storage.Transaction{
		UserID:        user.ID,
		WalletAddress: walletAddress,
		Kind:          kind,
		Status:        storage.TxStatusSuccess,
		TxHash:        txHash,
		Detail:        detail,
	})
}
