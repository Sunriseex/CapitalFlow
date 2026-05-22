package config

import "testing"

func TestInitLoadsTrustedProxies(t *testing.T) {
	oldConfig := AppConfig
	t.Cleanup(func() {
		AppConfig = oldConfig
	})

	t.Setenv("CAPITALFLOW_ENV_FILE", "missing-test-env-file")
	t.Setenv("TRUSTED_PROXIES", "192.0.2.10, 2001:db8::/32")

	if err := Init(); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if len(AppConfig.TrustedProxies) != 2 {
		t.Fatalf("trusted proxies = %v, want 2 entries", AppConfig.TrustedProxies)
	}
	if AppConfig.TrustedProxies[0] != "192.0.2.10" || AppConfig.TrustedProxies[1] != "2001:db8::/32" {
		t.Fatalf("trusted proxies = %v", AppConfig.TrustedProxies)
	}
}

func TestValidateAuthSecretRequiresMinimumLength(t *testing.T) {
	if err := ValidateAuthSecret("JWT_SECRET", "short"); err == nil {
		t.Fatal("expected short secret error")
	}
	if err := ValidateAuthSecret("JWT_SECRET", "01234567890123456789012345678901"); err != nil {
		t.Fatalf("valid secret: %v", err)
	}
}
