package utils

import (
	"unicode"

	apperr "github.com/rekall/backend/pkg/errors"
)

// ValidatePassword enforces minimum password complexity.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return apperr.Unprocessable("password must be at least 8 characters", nil)
	}
	hasLetter, hasDigit := false, false
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return apperr.Unprocessable("password must contain at least one letter and one digit", nil)
	}
	return nil
}
