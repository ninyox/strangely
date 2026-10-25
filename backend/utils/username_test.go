package utils

import (
	"strings"
	"testing"
)

func TestNormalizeUsernameAccepts(t *testing.T) {
	cases := map[string]string{
		"  ada  ":        "ada",
		"Ada   Lovelace": "Ada Lovelace",
		"jo-jo_99":       "jo-jo_99",
		"Ada.L":          "Ada.L",
		"李雷雷":            "李雷雷",
	}

	for input, want := range cases {
		got, err := NormalizeUsername(input)
		if err != nil {
			t.Fatalf("NormalizeUsername(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Errorf("NormalizeUsername(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeUsernameRejects(t *testing.T) {
	cases := map[string]error{
		"":                          ErrUsernameEmpty,
		"   ":                       ErrUsernameEmpty,
		"ab":                        ErrUsernameTooShort,
		"a\tb":                      ErrUsernameInvalidChars,
		"<script>":                  ErrUsernameInvalidChars,
		"ada\nlovelace":             ErrUsernameInvalidChars,
		"thisnameiswaytoolongforus": ErrUsernameTooLong,
	}

	for input, want := range cases {
		if _, err := NormalizeUsername(input); err != want {
			t.Errorf("NormalizeUsername(%q) error = %v, want %v", input, err, want)
		}
	}
}

func TestNormalizeUsernameCountsRunes(t *testing.T) {
	// Twenty accented characters are twenty runes and thirty bytes; the limit
	// is about how the name reads, not how it is encoded.
	if _, err := NormalizeUsername(strings.Repeat("é", UsernameMaxLength)); err != nil {
		t.Fatalf("expected 20 rune name to be accepted, got %v", err)
	}
	if _, err := NormalizeUsername(strings.Repeat("é", UsernameMaxLength+1)); err != ErrUsernameTooLong {
		t.Fatalf("expected 21 rune name to be rejected, got %v", err)
	}
}

func TestSanitizeNote(t *testing.T) {
	got := SanitizeNote("  rude \x00\x07 person \n here ")
	if got != "rude person here" {
		t.Errorf("SanitizeNote stripped/collapsed incorrectly: %q", got)
	}

	long := strings.Repeat("x", NoteMaxLength+50)
	if runes := []rune(SanitizeNote(long)); len(runes) != NoteMaxLength {
		t.Errorf("SanitizeNote length = %d, want %d", len(runes), NoteMaxLength)
	}
}
