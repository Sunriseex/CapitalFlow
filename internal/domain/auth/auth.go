package auth

import (
	"fmt"
	"net/mail"
	"strings"

	zxcvbn "github.com/nbutton23/zxcvbn-go"
)

func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", fmt.Errorf("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", fmt.Errorf("invalid email")
	}
	return email, nil
}

func ValidatePassword(password, email string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}
	if zxcvbn.PasswordStrength(password, passwordUserInputs(email)).Score < 3 {
		return fmt.Errorf("password is too weak")
	}
	return nil
}

func passwordUserInputs(email string) []string {
	inputs := []string{email}
	local, domain, found := strings.Cut(email, "@")
	if found {
		inputs = append(inputs, local, domain)
	}
	return inputs
}
