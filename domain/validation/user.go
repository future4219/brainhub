package validation

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidPassword = errors.New("password must be 12 to 72 bytes")
	ErrInvalidName     = errors.New("name must be 1 to 80 characters without control characters")
)

func NormalizeEmail(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if len(normalized) == 0 || len(normalized) > 254 {
		return "", ErrInvalidEmail
	}
	address, err := mail.ParseAddress(normalized)
	if err != nil || address.Address != normalized || !strings.Contains(normalized, "@") {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

func ValidatePassword(value string) error {
	if len(value) < 12 || len(value) > 72 {
		return ErrInvalidPassword
	}
	return nil
}

func NormalizeName(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	if utf8.RuneCountInString(normalized) < 1 || utf8.RuneCountInString(normalized) > 80 {
		return "", ErrInvalidName
	}
	for _, r := range normalized {
		if unicode.IsControl(r) {
			return "", ErrInvalidName
		}
	}
	return normalized, nil
}
