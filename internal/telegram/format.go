package telegram

import (
	"fmt"
	"math/big"
	"strings"
)

// weiToDecimalString renders a base-unit integer amount as a human-readable
// decimal string with the given number of decimals, trimming trailing
// zeros.
func weiToDecimalString(amount *big.Int, decimals uint8) string {
	if amount == nil {
		return "0"
	}
	neg := amount.Sign() < 0
	abs := new(big.Int).Abs(amount)

	div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int)
	frac := new(big.Int)
	whole.DivMod(abs, div, frac)

	fracStr := frac.String()
	if pad := int(decimals) - len(fracStr); pad > 0 {
		fracStr = strings.Repeat("0", pad) + fracStr
	}
	fracStr = strings.TrimRight(fracStr, "0")

	out := whole.String()
	if fracStr != "" {
		out += "." + fracStr
	}
	if neg {
		out = "-" + out
	}
	return out
}

// decimalStringToWei parses a human-entered decimal amount string into a
// base-unit integer using the given number of decimals.
func decimalStringToWei(amount string, decimals uint8) (*big.Int, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return nil, fmt.Errorf("amount is required")
	}
	neg := strings.HasPrefix(amount, "-")
	if neg {
		return nil, fmt.Errorf("amount must be positive")
	}

	parts := strings.SplitN(amount, ".", 2)
	wholePart := parts[0]
	if wholePart == "" {
		wholePart = "0"
	}
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}
	if len(fracPart) > int(decimals) {
		return nil, fmt.Errorf("amount has more than %d decimal places", decimals)
	}
	fracPart = fracPart + strings.Repeat("0", int(decimals)-len(fracPart))

	combined := wholePart + fracPart
	value, ok := new(big.Int).SetString(combined, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount %q", amount)
	}
	return value, nil
}

// shortAddr abbreviates an address for compact display: 0x1234…abcd.
func shortAddr(addr string) string {
	if len(addr) <= 12 {
		return addr
	}
	return addr[:6] + "…" + addr[len(addr)-4:]
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
		"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
	)
	return replacer.Replace(s)
}
