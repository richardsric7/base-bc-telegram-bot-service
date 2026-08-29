package telegram

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/richardsric7/base-bc-telegram-bot-service/internal/storage"
)

// maxCodeGenAttempts bounds retries if a freshly generated code collides
// with an existing one (astronomically unlikely at 6 digits, but cheap to
// guard against).
const maxCodeGenAttempts = 5

func (b *Bot) cmdGenerateCode(user *storage.User, chatID int64) {
	if user.Role != storage.RoleAdmin {
		b.reply(chatID, "Only the admin can generate invite codes.")
		return
	}

	var (
		code string
		err  error
	)
	for attempt := 0; attempt < maxCodeGenAttempts; attempt++ {
		code, err = generateNumericCode()
		if err != nil {
			b.reply(chatID, "Failed to generate a code: "+err.Error())
			return
		}
		_, createErr := b.store.CreateInviteCode(code, user.ID, time.Now().Add(inviteCodeTTL))
		if createErr == nil {
			botUsername := b.api.Self.UserName
			link := ""
			if botUsername != "" {
				link = fmt.Sprintf("\nInvite link: https://t.me/%s?start=%s", botUsername, code)
			}
			b.reply(chatID, fmt.Sprintf(
				"✅ New invite code: %s\nValid for %s. Share it with the new user — they can send it to me as a message or use the link below.%s",
				code, inviteCodeTTL, link,
			))
			return
		}
		// Retry only on a unique-constraint collision; anything else is a
		// real failure worth surfacing immediately.
		if !isUniqueConstraintErr(createErr) {
			b.reply(chatID, "Failed to save the invite code: "+createErr.Error())
			return
		}
	}
	b.reply(chatID, "Failed to generate a unique invite code, please try again.")
}

func (b *Bot) cmdListCodes(user *storage.User, chatID int64) {
	if user.Role != storage.RoleAdmin {
		b.reply(chatID, "Only the admin can view invite codes.")
		return
	}
	codes, err := b.store.ListActiveInviteCodes(user.ID)
	if err != nil {
		b.reply(chatID, "Failed to list invite codes: "+err.Error())
		return
	}
	if len(codes) == 0 {
		b.reply(chatID, "No active invite codes. Use /generate_code to create one.")
		return
	}
	var sb strings.Builder
	sb.WriteString("Active invite codes:\n")
	for _, c := range codes {
		sb.WriteString(fmt.Sprintf("• %s — expires %s\n", c.Code, c.ExpiresAt.Format(time.RFC822)))
	}
	b.reply(chatID, sb.String())
}

func (b *Bot) cmdRevokeCode(user *storage.User, chatID int64, args string) {
	if user.Role != storage.RoleAdmin {
		b.reply(chatID, "Only the admin can revoke invite codes.")
		return
	}
	code := strings.TrimSpace(args)
	if !sixDigitCode.MatchString(code) {
		b.reply(chatID, "Usage: /revoke_code <6-digit code>")
		return
	}
	if err := b.store.RevokeInviteCode(code); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			b.reply(chatID, "No active code matching that value (it may already be used or expired).")
			return
		}
		b.reply(chatID, "Failed to revoke code: "+err.Error())
		return
	}
	b.reply(chatID, "✅ Revoked code "+code+".")
}

// isUniqueConstraintErr reports whether err looks like a unique-constraint
// violation from the SQLite driver, without depending on driver-specific
// error types.
func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}
