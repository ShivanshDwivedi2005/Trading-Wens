package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address            string
	FrontendURL        string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	SessionSecret      string
	SessionTTL         time.Duration
	AlpacaDataURL      string
	AlpacaTradingURL   string
	AlpacaAPIKeyID     string
	AlpacaAPISecretKey string
	AlpacaDataFeed     string
	GDELTAPIURL        string
	CORSAllowedOrigins []string
}

func Load() (Config, error) {
	if err := loadLocalEnvironment(); err != nil {
		return Config{}, err
	}

	cfg := Config{
		Address:            valueOrDefault("BACKEND_ADDRESS", ":8080"),
		FrontendURL:        strings.TrimRight(valueOrDefault("FRONTEND_URL", "http://localhost:3000"), "/"),
		GoogleClientID:     strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		GoogleRedirectURL:  strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URI")),
		SessionSecret:      strings.TrimSpace(os.Getenv("JWT_ACCESS_SECRET")),
		AlpacaDataURL:      strings.TrimRight(valueOrDefault("ALPACA_DATA_REST_URL", "https://data.alpaca.markets"), "/"),
		AlpacaTradingURL:   valueOrDefault("ALPACA_TRADING_REST_URL", "https://paper-api.alpaca.markets"),
		AlpacaAPIKeyID:     strings.TrimSpace(os.Getenv("ALPACA_API_KEY_ID")),
		AlpacaAPISecretKey: strings.TrimSpace(os.Getenv("ALPACA_API_SECRET_KEY")),
		AlpacaDataFeed:     valueOrDefault("ALPACA_DATA_FEED", "iex"),
		GDELTAPIURL:        strings.TrimRight(valueOrDefault("GDELT_API_URL", "https://api.gdeltproject.org/api/v2/doc/doc"), "/"),
		CORSAllowedOrigins: splitList(valueOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")),
	}

	var missing []string
	if cfg.GoogleClientID == "" {
		missing = append(missing, "GOOGLE_CLIENT_ID")
	}
	if cfg.GoogleClientSecret == "" {
		missing = append(missing, "GOOGLE_CLIENT_SECRET")
	}
	if cfg.GoogleRedirectURL == "" {
		missing = append(missing, "GOOGLE_REDIRECT_URI")
	}
	if cfg.SessionSecret == "" {
		missing = append(missing, "JWT_ACCESS_SECRET")
	}
	if cfg.AlpacaAPIKeyID == "" {
		missing = append(missing, "ALPACA_API_KEY_ID")
	}
	if cfg.AlpacaAPISecretKey == "" {
		missing = append(missing, "ALPACA_API_SECRET_KEY")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if len(cfg.SessionSecret) < 32 {
		return Config{}, errors.New("JWT_ACCESS_SECRET must contain at least 32 characters")
	}
	if !validHTTPURL(cfg.FrontendURL) {
		return Config{}, errors.New("FRONTEND_URL must be an HTTP or HTTPS URL")
	}
	if !validHTTPURL(cfg.GoogleRedirectURL) {
		return Config{}, errors.New("GOOGLE_REDIRECT_URI must be an HTTP or HTTPS URL")
	}
	if !strings.HasPrefix(cfg.AlpacaDataURL, "https://") && !strings.HasPrefix(cfg.AlpacaDataURL, "http://") {
		return Config{}, errors.New("ALPACA_DATA_REST_URL must be an HTTP or HTTPS URL")
	}
	normalizedTradingURL, err := normalizeAlpacaTradingURL(cfg.AlpacaTradingURL)
	if err != nil {
		return Config{}, err
	}
	cfg.AlpacaTradingURL = normalizedTradingURL
	if !strings.HasPrefix(cfg.GDELTAPIURL, "https://") && !strings.HasPrefix(cfg.GDELTAPIURL, "http://") {
		return Config{}, errors.New("GDELT_API_URL must be an HTTP or HTTPS URL")
	}
	if !validAlpacaFeed(cfg.AlpacaDataFeed) {
		return Config{}, fmt.Errorf("unsupported ALPACA_DATA_FEED %q", cfg.AlpacaDataFeed)
	}
	sessionTTLSeconds, err := strconv.Atoi(valueOrDefault("AUTH_SESSION_TTL_SECONDS", "28800"))
	if err != nil || sessionTTLSeconds < 300 || sessionTTLSeconds > 604800 {
		return Config{}, errors.New("AUTH_SESSION_TTL_SECONDS must be between 300 and 604800")
	}
	cfg.SessionTTL = time.Duration(sessionTTLSeconds) * time.Second

	return cfg, nil
}

func validHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func normalizeAlpacaTradingURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host != "paper-api.alpaca.markets" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("ALPACA_TRADING_REST_URL must use the Alpaca paper trading endpoint")
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path != "" && path != "/v2" {
		return "", errors.New("ALPACA_TRADING_REST_URL must use the Alpaca paper trading endpoint")
	}
	return "https://paper-api.alpaca.markets", nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func firstValue(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, strings.TrimRight(trimmed, "/"))
		}
	}
	return values
}

func validAlpacaFeed(feed string) bool {
	switch feed {
	case "iex", "sip", "delayed_sip", "boats", "overnight", "otc":
		return true
	default:
		return false
	}
}

func loadLocalEnvironment() error {
	paths := []string{
		filepath.Join("frontend", ".env"),
		filepath.Join("..", "frontend", ".env"),
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("backend", ".env"),
	}
	for _, path := range paths {
		file, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("open local environment file: %w", err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			name, value, found := strings.Cut(line, "=")
			name = strings.TrimSpace(name)
			if !found || name == "" {
				continue
			}
			value = strings.TrimSpace(value)
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			if _, exists := os.LookupEnv(name); !exists {
				if err := os.Setenv(name, value); err != nil {
					return fmt.Errorf("load local environment variable %s: %w", name, err)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			return fmt.Errorf("read local environment file: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close local environment file: %w", err)
		}
	}
	return nil
}
