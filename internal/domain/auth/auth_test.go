package auth

import "testing"

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail(" USER@example.COM ")
	if err != nil {
		t.Fatalf("normalize email: %v", err)
	}
	if got != "user@example.com" {
		t.Fatalf("email = %q, want user@example.com", got)
	}
	if _, err := NormalizeEmail("not email"); err == nil {
		t.Fatal("expected invalid email error")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("correct horse battery staple 2026", "user@example.com"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
	if err := ValidatePassword("short", "user@example.com"); err == nil {
		t.Fatal("expected short password error")
	}
	if err := ValidatePassword("user@example.com", "user@example.com"); err == nil {
		t.Fatal("expected weak password error")
	}
}
