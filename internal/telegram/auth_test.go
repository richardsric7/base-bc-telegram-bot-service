package telegram

import "testing"

func TestGenerateNumericCodeIsSixDigits(t *testing.T) {
	for i := 0; i < 200; i++ {
		code, err := generateNumericCode()
		if err != nil {
			t.Fatalf("generateNumericCode: %v", err)
		}
		if !sixDigitCode.MatchString(code) {
			t.Fatalf("generateNumericCode() = %q, want 6 digits", code)
		}
	}
}

func TestSixDigitCodeRegex(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"123456", true},
		{"000000", true},
		{"12345", false},
		{"1234567", false},
		{"12345a", false},
		{"", false},
		{" 123456", false},
	}
	for _, c := range cases {
		if got := sixDigitCode.MatchString(c.in); got != c.want {
			t.Errorf("sixDigitCode.MatchString(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
