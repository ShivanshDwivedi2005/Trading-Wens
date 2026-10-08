package config

import (
	"os"
	"testing"
)

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

func TestXConfigurationRequiresTokenWhenEnabled(t *testing.T) {
	keys := []string{
		"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_REDIRECT_URI", "JWT_ACCESS_SECRET",
		"ALPACA_API_KEY_ID", "ALPACA_API_SECRET_KEY", "X_ENABLED", "X_BEARER_TOKEN",
	}
	previous := make(map[string]string, len(keys))
	for _, key := range keys {
		previous[key] = os.Getenv(key)
	}
	t.Cleanup(func() {
		for _, key := range keys {
			_ = os.Setenv(key, previous[key])
		}
	})
	values := map[string]string{
		"GOOGLE_CLIENT_ID": "client", "GOOGLE_CLIENT_SECRET": "secret",
		"GOOGLE_REDIRECT_URI": "http://localhost:8080/auth/google/callback",
		"JWT_ACCESS_SECRET":   "12345678901234567890123456789012",
		"ALPACA_API_KEY_ID":   "key", "ALPACA_API_SECRET_KEY": "secret",
		"X_ENABLED": "true", "X_BEARER_TOKEN": "",
	}
	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Load()
	if err == nil || err.Error() != "X_BEARER_TOKEN is required when X_ENABLED is true" {
		t.Fatalf("unexpected error: %v", err)
	}
}
