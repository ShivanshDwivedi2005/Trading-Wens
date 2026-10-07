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
