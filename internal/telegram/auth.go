package telegram

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/richardsric7/base-bc-telegram-bot-service/internal/storage"
)

// inviteCodeTTL is how long an admin-issued invite code stays redeemable.
const inviteCodeTTL = 24 * time.Hour

var sixDigitCode = regexp.MustCompile(`^\d{6}$`)

// generateNumericCode returns a cryptographically random 6-digit code,
// zero-padded (e.g. "004821").
func generateNumericCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// resolveUser looks up the registered user for a Telegram sender, or nil if
// they haven't registered yet. It also keeps the cached username fresh.
func (b *Bot) resolveUser(from *tgbotapi.User) (*storage.User, error) {
	user, err := b.store.GetUserByTelegramID(from.ID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	b.store.TouchUsername(user, from.UserName)
	return user, nil
}

// handleUnregistered handles a message from a Telegram user who has not
// registered yet. It looks for a 6-digit code either as /start's argument
// (so an invite link like t.me/<bot>?start=123456 works) or as the message
// text itself, and attempts to register the sender with it. Absent a code,
// it explains what to do next.
func (b *Bot) handleUnregistered(ctx context.Context, msg *tgbotapi.Message) {
	code := ""
	switch {
	case msg.IsCommand() && msg.Command() == "start" && sixDigitCode.MatchString(strings.TrimSpace(msg.CommandArguments())):
		code = strings.TrimSpace(msg.CommandArguments())
	case !msg.IsCommand() && sixDigitCode.MatchString(strings.TrimSpace(msg.Text)):
		code = strings.TrimSpace(msg.Text)
	}

	if code == "" {
		b.sendRegistrationPrompt(msg.Chat.ID)
		return
	}
	b.attemptRegistration(ctx, msg, code)
}

func (b *Bot) sendRegistrationPrompt(chatID int64) {
	hasAdmin, err := b.store.HasAdmin()
	if err != nil {
		b.reply(chatID, "Internal error checking registration state. Please try again.")
		return
	}
	if !hasAdmin {
		b.reply(chatID, "👋 This bot has no admin yet. Send the 6-digit admin setup code to claim it.")
		return
	}
	b.reply(chatID, "👋 You're not registered yet. Ask the admin for a 6-digit invite code, then send it to me (or open the invite link they give you).")
}

func (b *Bot) attemptRegistration(ctx context.Context, msg *tgbotapi.Message, code string) {
	chatID := msg.Chat.ID
	from := msg.From

	hasAdmin, err := b.store.HasAdmin()
	if err != nil {
		b.reply(chatID, "Internal error checking registration state. Please try again.")
		return
	}

	if !hasAdmin {
		if subtle.ConstantTimeCompare([]byte(code), []byte(b.cfg.AdminSetupCode)) != 1 {
			b.reply(chatID, "❌ Invalid admin setup code.")
			return
		}
		if _, err := b.store.CreateUser(from.ID, from.UserName, storage.RoleAdmin); err != nil {
			b.reply(chatID, "Failed to register you as admin: "+err.Error())
			return
		}
		b.reply(chatID, "✅ You're registered as the admin of this bot. Send /help to see what you can do, or /generate_code to invite other users.")
		return
	}

	invite, err := b.store.GetActiveInviteCode(code)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			b.reply(chatID, "❌ That code is invalid, expired, or already used. Ask the admin for a new one.")
			return
		}
		b.reply(chatID, "Internal error validating that code. Please try again.")
		return
	}

	user, err := b.store.CreateUser(from.ID, from.UserName, storage.RoleUser)
	if err != nil {
		b.reply(chatID, "Failed to register you: "+err.Error())
		return
	}
	if err := b.store.MarkInviteCodeUsed(invite.ID, user.ID); err != nil {
		// The user account was created; log-and-continue rather than leave
		// them stuck, but surface that the code bookkeeping failed.
		b.reply(chatID, "✅ You're registered, but marking the invite code used failed — let the admin know.")
		return
	}
	b.reply(chatID, "✅ You're registered. Send /help to see what you can do.")
}
