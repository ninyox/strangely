package utils

import (
	"errors"
	"strings"
	"unicode"
)

const (
	// UsernameMinLength and UsernameMaxLength are counted in runes, not bytes,
	// so a name made of multi-byte characters is measured the way a person
	// reads it.
	UsernameMinLength = 3
	UsernameMaxLength = 20
	// NoteMaxLength caps the free text a report may carry.
	NoteMaxLength = 500
)

var (
	ErrUsernameEmpty        = errors.New("Please choose a name.")
	ErrUsernameTooShort     = errors.New("Names need at least 3 characters.")
	ErrUsernameTooLong      = errors.New("Names can be at most 20 characters.")
	ErrUsernameInvalidChars = errors.New("Use letters, numbers, spaces, and . _ - only.")
)

// NormalizeUsername turns raw visitor input into the display name the rest of
// the app uses. It trims, collapses runs of whitespace, rejects control
// characters and punctuation that could be used to spoof markup, and enforces
// the length window. The same rules live in the frontend (lib/username.ts) so
// the form can explain the problem before a request is made; the backend is
// still the authority.
func NormalizeUsername(raw string) (string, error) {
	if strings.ContainsFunc(raw, unicode.IsControl) {
		return "", ErrUsernameInvalidChars
	}

	fields := strings.Fields(raw)
	name := strings.Join(fields, " ")
	if name == "" {
		return "", ErrUsernameEmpty
	}

	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), unicode.IsSpace(r):
		case r == '.', r == '_', r == '-', r == '\'':
		default:
			return "", ErrUsernameInvalidChars
		}
	}

	if len([]rune(name)) < UsernameMinLength {
		return "", ErrUsernameTooShort
	}
	if len([]rune(name)) > UsernameMaxLength {
		return "", ErrUsernameTooLong
	}

	return name, nil
}

// SanitizeNote cleans free text attached to a report: control characters are
// stripped (they are never meaningful here), whitespace is collapsed, and the
// result is capped. Notes are stored and never rendered as HTML.
func SanitizeNote(raw string) string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, raw)

	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if len([]rune(cleaned)) > NoteMaxLength {
		cleaned = string([]rune(cleaned)[:NoteMaxLength])
	}

	return strings.TrimSpace(cleaned)
}
