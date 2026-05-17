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
