package config

import "testing"

func TestNormalizeAlpacaTradingURL(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  string
		wantError bool
	}{
		{name: "host", value: "https://paper-api.alpaca.markets", expected: "https://paper-api.alpaca.markets"},
		{name: "versioned", value: "https://paper-api.alpaca.markets/v2", expected: "https://paper-api.alpaca.markets"},
		{name: "trailing slash", value: "https://paper-api.alpaca.markets/v2/", expected: "https://paper-api.alpaca.markets"},
		{name: "live endpoint", value: "https://api.alpaca.markets", wantError: true},
		{name: "unexpected path", value: "https://paper-api.alpaca.markets/v1", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := normalizeAlpacaTradingURL(test.value)
			if test.wantError {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalize URL: %v", err)
			}
			if actual != test.expected {
				t.Fatalf("expected %q, got %q", test.expected, actual)
			}
		})
	}
}

func TestBlueskyConfigurationIsKeylessAndEnabledByDefault(t *testing.T) {
	values := map[string]string{
		"DATABASE_URL":           "postgresql://user:password@host.neon.tech/trading?sslmode=require",
		"GOOGLE_OAUTH_CLIENT_ID": "client", "GOOGLE_OAUTH_CLIENT_SECRET": "secret",
		"GOOGLE_OAUTH_REDIRECT_URI": "http://localhost:8080/auth/google/callback",
		"APP_SESSION_SIGNING_KEY":   "12345678901234567890123456789012",
		"ALPACA_API_KEY":            "key", "ALPACA_SECRET_KEY": "secret",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	t.Setenv("BLUESKY_API_ENABLED", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.BlueskyEnabled || cfg.BlueskyAPIURL != "https://public.api.bsky.app" {
		t.Fatalf("unexpected Bluesky configuration: %#v", cfg)
	}
}

func TestCanonicalEnvironmentNames(t *testing.T) {
	values := map[string]string{
		"DATABASE_URL":           "postgresql://user:password@host.neon.tech/trading?sslmode=require",
		"GOOGLE_OAUTH_CLIENT_ID": "client", "GOOGLE_OAUTH_CLIENT_SECRET": "secret",
		"GOOGLE_OAUTH_REDIRECT_URI": "http://localhost:8080/auth/google/callback",
		"APP_SESSION_SIGNING_KEY":   "12345678901234567890123456789012",
		"ALPACA_API_KEY":            "key", "ALPACA_SECRET_KEY": "secret",
		"BLUESKY_API_ENABLED": "false",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseMaxConns != 10 || cfg.AlpacaAPIKeyID != "key" {
		t.Fatalf("canonical variables were not loaded: %#v", cfg)
	}
}

func TestValidPostgresURL(t *testing.T) {
	if !validPostgresURL("postgresql://user:password@host.neon.tech/trading?sslmode=require") {
		t.Fatal("expected Neon connection string to be valid")
	}
	if validPostgresURL("https://database.example.com") {
		t.Fatal("expected HTTP database URL to be rejected")
	}
}
