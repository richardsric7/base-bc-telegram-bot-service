package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/richardsric7/base-bc-telegram-bot-service/internal/storage"
	"github.com/richardsric7/base-bc-telegram-bot-service/internal/swap"
)

// resolveSwapToken maps the literal "ETH" to the configured WETH address
// (used for both quoting and as the router's native-input token), or parses
// a hex ERC-20 address.
func (b *Bot) resolveSwapToken(s string) (addr common.Address, native bool, err error) {
	if strings.EqualFold(s, "ETH") {
		return b.swap.WETH(), true, nil
	}
	if !common.IsHexAddress(s) {
		return common.Address{}, false, fmt.Errorf("%q is not \"ETH\" or a valid token address", s)
	}
	return common.HexToAddress(s), false, nil
}

func (b *Bot) tokenDecimals(ctx context.Context, addr common.Address, native bool) uint8 {
	if native {
		return 18
	}
	info, err := b.token.Info(ctx, addr)
	if err != nil {
		return 18
	}
	return info.Decimals
}

func (b *Bot) cmdSwapQuote(ctx context.Context, chatID int64, args string) {
	parts := strings.Fields(args)
	if len(parts) != 3 {
		b.reply(chatID, "Usage: /swap_quote <fromToken|ETH> <toToken|ETH> <amount>")
		return
	}
	fromSym, toSym, amountStr := parts[0], parts[1], parts[2]

	fromAddr, _, err := b.resolveSwapToken(fromSym)
	if err != nil {
		b.reply(chatID, err.Error())
		return
	}
	toAddr, _, err := b.resolveSwapToken(toSym)
	if err != nil {
		b.reply(chatID, err.Error())
		return
	}

	fromDecimals := b.tokenDecimals(ctx, fromAddr, strings.EqualFold(fromSym, "ETH"))
	amountIn, err := decimalStringToWei(amountStr, fromDecimals)
	if err != nil {
		b.reply(chatID, "Invalid amount: "+err.Error())
		return
	}

	quoted, err := b.swap.Quote(ctx, fromAddr, toAddr, amountIn, swap.DefaultFeeTier)
	if err != nil {
		b.reply(chatID, "Quote failed: "+err.Error())
		return
	}
	toDecimals := b.tokenDecimals(ctx, toAddr, strings.EqualFold(toSym, "ETH"))
	b.reply(chatID, fmt.Sprintf("Quote: %s %s ≈ %s %s\n(0.3%% pool fee tier, no slippage applied)", amountStr, fromSym, weiToDecimalString(quoted, toDecimals), toSym))
}

func (b *Bot) cmdSwapExecute(ctx context.Context, user *storage.User, chatID, telegramUserID int64, args string) {
	parts := strings.Fields(args)
	if len(parts) != 5 {
		b.reply(chatID, "Usage: /swap_execute <wallet> <fromToken|ETH> <toToken|ETH> <amount> <slippageBps>")
		return
	}
	walletLabel, fromSym, toSym, amountStr, slippageStr := parts[0], parts[1], parts[2], parts[3], parts[4]

	w, err := b.wallet.Get(user.ID, walletLabel)
	if err != nil {
		b.reply(chatID, walletErrorMessage(err))
		return
	}
	slippageBps, err := strconv.Atoi(slippageStr)
	if err != nil || slippageBps < 0 || slippageBps > 5000 {
		b.reply(chatID, "slippageBps must be a whole number between 0 and 5000 (50%).")
		return
	}

	fromAddr, fromNative, err := b.resolveSwapToken(fromSym)
	if err != nil {
		b.reply(chatID, err.Error())
		return
	}
	toAddr, _, err := b.resolveSwapToken(toSym)
	if err != nil {
		b.reply(chatID, err.Error())
		return
	}

	fromDecimals := b.tokenDecimals(ctx, fromAddr, fromNative)
	amountIn, err := decimalStringToWei(amountStr, fromDecimals)
	if err != nil {
		b.reply(chatID, "Invalid amount: "+err.Error())
		return
	}

	quoted, err := b.swap.Quote(ctx, fromAddr, toAddr, amountIn, swap.DefaultFeeTier)
	if err != nil {
		b.reply(chatID, "Quote failed: "+err.Error())
		return
	}
	minOut := swap.MinAmountOut(quoted, uint(slippageBps))
	toDecimals := b.tokenDecimals(ctx, toAddr, strings.EqualFold(toSym, "ETH"))

	description := fmt.Sprintf(
		"Swap %s %s → %s\nWallet: %s (%s)\nExpected out: ≈%s %s\nMin out (after %.2f%% slippage): %s %s",
		amountStr, fromSym, toSym, w.Label, w.Address,
		weiToDecimalString(quoted, toDecimals), toSym,
		float64(slippageBps)/100,
		weiToDecimalString(minOut, toDecimals), toSym,
	)

	b.confirm(chatID, telegramUserID, description, func() (string, error) {
		key, err := b.wallet.PrivateKey(w)
		if err != nil {
			return "", err
		}

		if !fromNative {
			approveReceipt, err := b.swap.EnsureAllowance(ctx, key, fromAddr, amountIn)
			if err != nil {
				return "", fmt.Errorf("approve failed: %w", err)
			}
			if approveReceipt != nil {
				b.logTx(user, w.Address, storage.TxKindApprove, approveReceipt.TxHash.Hex(), "Approve router for swap")
			}
		}

		result, err := b.swap.Execute(ctx, key, swap.ExecuteParams{
			TokenIn:          fromAddr,
			TokenOut:         toAddr,
			AmountIn:         amountIn,
			AmountOutMinimum: minOut,
			FeeTier:          swap.DefaultFeeTier,
			Native:           fromNative,
		})
		if err != nil {
			return "", err
		}
		b.logTx(user, w.Address, storage.TxKindSwap, result.Tx.Hash().Hex(), description)
		return b.chain.TxURL(result.Tx.Hash().Hex()), nil
	})
}
