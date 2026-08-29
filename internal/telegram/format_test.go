package telegram

import (
	"math/big"
	"testing"
)

func TestWeiToDecimalString(t *testing.T) {
	cases := []struct {
		amount   string
		decimals uint8
		want     string
	}{
		{"1000000000000000000", 18, "1"},
		{"1500000000000000000", 18, "1.5"},
		{"0", 18, "0"},
		{"1", 18, "0.000000000000000001"},
		{"123456", 6, "0.123456"},
	}
	for _, c := range cases {
		amount, ok := new(big.Int).SetString(c.amount, 10)
		if !ok {
			t.Fatalf("bad test input %q", c.amount)
		}
		got := weiToDecimalString(amount, c.decimals)
		if got != c.want {
			t.Errorf("weiToDecimalString(%s, %d) = %q, want %q", c.amount, c.decimals, got, c.want)
		}
	}
}

func TestDecimalStringToWei(t *testing.T) {
	cases := []struct {
		amount   string
		decimals uint8
		want     string
	}{
		{"1", 18, "1000000000000000000"},
		{"1.5", 18, "1500000000000000000"},
		{"0.123456", 6, "123456"},
		{"0", 18, "0"},
	}
	for _, c := range cases {
		got, err := decimalStringToWei(c.amount, c.decimals)
		if err != nil {
			t.Fatalf("decimalStringToWei(%s, %d): %v", c.amount, c.decimals, err)
		}
		if got.String() != c.want {
			t.Errorf("decimalStringToWei(%s, %d) = %s, want %s", c.amount, c.decimals, got.String(), c.want)
		}
	}
}

func TestDecimalStringToWeiRejectsTooManyDecimals(t *testing.T) {
	if _, err := decimalStringToWei("1.23", 1); err == nil {
		t.Fatal("expected error for too many decimal places")
	}
}

func TestDecimalStringToWeiRejectsNegative(t *testing.T) {
	if _, err := decimalStringToWei("-1", 18); err == nil {
		t.Fatal("expected error for negative amount")
	}
}
