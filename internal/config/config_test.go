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

func TestInitParsesSecurityEnv(t *testing.T) {
	oldConfig := AppConfig
	t.Cleanup(func() {
		AppConfig = oldConfig
	})

	t.Setenv("CAPITALFLOW_ENV_FILE", "missing-test-env-file")
	t.Setenv("APP_ENV", "production")
	t.Setenv("PUBLIC_ORIGIN", "https://capitalflow.home.arpa")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_SAMESITE", "Lax")
	t.Setenv("ALLOW_DIRECT_IP_LOGIN", "false")

	if err := Init(); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if AppConfig.AppEnv != "production" {
		t.Fatalf("app env = %q", AppConfig.AppEnv)
	}
	if AppConfig.PublicOrigin != "https://capitalflow.home.arpa" {
		t.Fatalf("public origin = %q", AppConfig.PublicOrigin)
	}
	if AppConfig.PublicOriginHost != "capitalflow.home.arpa" {
		t.Fatalf("public origin host = %q", AppConfig.PublicOriginHost)
	}
	if !AppConfig.CookieSecure {
		t.Fatal("cookie secure = false, want true")
	}
	if AppConfig.CookieSameSite != "Lax" {
		t.Fatalf("cookie samesite = %q", AppConfig.CookieSameSite)
	}
	if AppConfig.AllowDirectIPLogin {
		t.Fatal("allow direct ip login = true, want false")
	}
}

func TestInitRejectsInvalidPublicOrigin(t *testing.T) {
	oldConfig := AppConfig
	t.Cleanup(func() {
		AppConfig = oldConfig
	})

	t.Setenv("CAPITALFLOW_ENV_FILE", "missing-test-env-file")
	t.Setenv("PUBLIC_ORIGIN", "capitalflow.home.arpa")

	if err := Init(); err == nil {
		t.Fatal("expected invalid PUBLIC_ORIGIN error")
	}
}

func TestInitRejectsInvalidAppEnv(t *testing.T) {
	oldConfig := AppConfig
	t.Cleanup(func() {
		AppConfig = oldConfig
	})

	t.Setenv("CAPITALFLOW_ENV_FILE", "missing-test-env-file")
	t.Setenv("APP_ENV", "prod")

	if err := Init(); err == nil {
		t.Fatal("expected invalid APP_ENV error")
	}
}

func TestInitNormalizesPublicOriginDefaultPort(t *testing.T) {
	oldConfig := AppConfig
	t.Cleanup(func() {
		AppConfig = oldConfig
	})

	t.Setenv("CAPITALFLOW_ENV_FILE", "missing-test-env-file")
	t.Setenv("PUBLIC_ORIGIN", "https://CapitalFlow.home.arpa:443")

	if err := Init(); err != nil {
		t.Fatalf("init config: %v", err)
	}
	if AppConfig.PublicOrigin != "https://capitalflow.home.arpa" {
		t.Fatalf("public origin = %q", AppConfig.PublicOrigin)
	}
	if AppConfig.PublicOriginHost != "capitalflow.home.arpa" {
		t.Fatalf("public origin host = %q", AppConfig.PublicOriginHost)
	}
}

func TestInitRejectsSameSiteNoneWithoutSecureCookie(t *testing.T) {
	oldConfig := AppConfig
	t.Cleanup(func() {
		AppConfig = oldConfig
	})

	t.Setenv("CAPITALFLOW_ENV_FILE", "missing-test-env-file")
	t.Setenv("COOKIE_SAMESITE", "None")
	t.Setenv("COOKIE_SECURE", "false")

	if err := Init(); err == nil {
		t.Fatal("expected SameSite=None without Secure error")
	}
}

func TestInitProductionRequiresPublicOriginAndStrongJWTSecret(t *testing.T) {
	tests := []struct {
		name         string
		publicOrigin string
		jwtSecret    string
	}{
		{name: "missing origin", jwtSecret: "01234567890123456789012345678901"},
		{name: "short secret", publicOrigin: "https://capitalflow.home.arpa", jwtSecret: "short"},
		{name: "placeholder secret", publicOrigin: "https://capitalflow.home.arpa", jwtSecret: "change-me-to-at-least-32-random-bytes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldConfig := AppConfig
			t.Cleanup(func() {
				AppConfig = oldConfig
			})

			t.Setenv("CAPITALFLOW_ENV_FILE", "missing-test-env-file")
			t.Setenv("APP_ENV", "production")
			t.Setenv("PUBLIC_ORIGIN", tt.publicOrigin)
			t.Setenv("JWT_SECRET", tt.jwtSecret)

			if err := Init(); err == nil {
				t.Fatal("expected production security validation error")
			}
		})
	}
}

func TestInitAllowsGeneratedSecretContainingWordSecret(t *testing.T) {
	oldConfig := AppConfig
	t.Cleanup(func() {
		AppConfig = oldConfig
	})

	t.Setenv("CAPITALFLOW_ENV_FILE", "missing-test-env-file")
	t.Setenv("APP_ENV", "production")
	t.Setenv("PUBLIC_ORIGIN", "https://capitalflow.home.arpa")
	t.Setenv("JWT_SECRET", "generated-secret-value-with-enough-random-bytes")

	if err := Init(); err != nil {
		t.Fatalf("init config: %v", err)
	}
}
