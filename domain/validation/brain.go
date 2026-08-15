package validation

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidBrainName        = errors.New("brain name must be 1 to 80 characters without control characters")
	ErrInvalidBrainDescription = errors.New("brain description must be at most 500 characters without control characters")
)

func ValidateBrainName(value string) error {
	if utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > 80 || hasControl(value) {
		return ErrInvalidBrainName
	}
	return nil
}

func ValidateBrainDescription(value string) error {
	if utf8.RuneCountInString(value) > 500 || hasControl(value) {
		return ErrInvalidBrainDescription
	}
	return nil
}

func hasControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
