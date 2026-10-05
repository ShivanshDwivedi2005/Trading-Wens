package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Address                string
	SupabaseURL            string
	SupabasePublishableKey string
}

func Load() (Config, error) {
	cfg := Config{
		Address:                valueOrDefault("BACKEND_ADDRESS", ":8080"),
		SupabaseURL:            strings.TrimRight(strings.TrimSpace(os.Getenv("SUPABASE_URL")), "/"),
		SupabasePublishableKey: strings.TrimSpace(os.Getenv("SUPABASE_PUBLISHABLE_KEY")),
	}

	var missing []string
	if cfg.SupabaseURL == "" {
		missing = append(missing, "SUPABASE_URL")
	}
	if cfg.SupabasePublishableKey == "" {
		missing = append(missing, "SUPABASE_PUBLISHABLE_KEY")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if !strings.HasPrefix(cfg.SupabaseURL, "https://") && !strings.HasPrefix(cfg.SupabaseURL, "http://") {
		return Config{}, errors.New("SUPABASE_URL must be an HTTP or HTTPS URL")
	}

	return cfg, nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
