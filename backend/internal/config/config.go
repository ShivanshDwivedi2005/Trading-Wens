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
	DatabaseURL        string
	DatabaseTimeout    time.Duration
	DatabaseMinConns   int32
	DatabaseMaxConns   int32
	GoogleEnabled      bool
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
	NLPAPIURL          string
	NLPRequestTimeout  time.Duration
	BlueskyEnabled     bool
	BlueskyAPIURL      string
	CORSAllowedOrigins []string
}

func Load() (Config, error) {
	if err := loadLocalEnvironment(); err != nil {
		return Config{}, err
	}

	cfg := Config{
		Address:            valueOrDefault("BACKEND_ADDRESS", ":8080"),
		FrontendURL:        strings.TrimRight(valueOrDefault("FRONTEND_URL", "http://localhost:3000"), "/"),
		DatabaseURL:        firstValue("DATABASE_URL"),
		GoogleClientID:     firstValue("GOOGLE_OAUTH_CLIENT_ID", "GOOGLE_CLIENT_ID"),
		GoogleClientSecret: firstValue("GOOGLE_OAUTH_CLIENT_SECRET", "GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  firstValue("GOOGLE_OAUTH_REDIRECT_URI", "GOOGLE_REDIRECT_URI"),
		SessionSecret:      firstValue("APP_SESSION_SIGNING_KEY", "JWT_ACCESS_SECRET"),
		AlpacaDataURL:      strings.TrimRight(valueOrDefault("ALPACA_DATA_REST_URL", "https://data.alpaca.markets"), "/"),
		AlpacaTradingURL:   valueOrDefault("ALPACA_TRADING_REST_URL", "https://paper-api.alpaca.markets"),
		AlpacaAPIKeyID:     firstValue("ALPACA_API_KEY", "ALPACA_API_KEY_ID"),
		AlpacaAPISecretKey: firstValue("ALPACA_SECRET_KEY", "ALPACA_API_SECRET", "ALPACA_API_SECRET_KEY"),
		AlpacaDataFeed:     valueOrDefault("ALPACA_DATA_FEED", "iex"),
		GDELTAPIURL:        strings.TrimRight(valueOrDefault("GDELT_API_URL", "https://api.gdeltproject.org/api/v2/doc/doc"), "/"),
		NLPAPIURL:          strings.TrimRight(firstValue("FINBERT_INFERENCE_URL", "NLP_API_URL"), "/"),
		BlueskyAPIURL:      strings.TrimRight(valueOrDefault("BLUESKY_API_BASE_URL", "https://public.api.bsky.app"), "/"),
		CORSAllowedOrigins: splitList(valueOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")),
	}
	if cfg.NLPAPIURL == "" {
		cfg.NLPAPIURL = "http://127.0.0.1:8090"
	}
	googleEnabledValue := firstValue("GOOGLE_OAUTH_ENABLED")
	if googleEnabledValue == "" {
		cfg.GoogleEnabled = cfg.GoogleClientID != "" || cfg.GoogleClientSecret != ""
	} else {
		googleEnabled, err := strconv.ParseBool(googleEnabledValue)
		if err != nil {
			return Config{}, errors.New("GOOGLE_OAUTH_ENABLED must be true or false")
		}
		cfg.GoogleEnabled = googleEnabled
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.GoogleEnabled && cfg.GoogleClientID == "" {
		missing = append(missing, "GOOGLE_OAUTH_CLIENT_ID")
	}
	if cfg.GoogleEnabled && cfg.GoogleClientSecret == "" {
		missing = append(missing, "GOOGLE_OAUTH_CLIENT_SECRET")
	}
	if cfg.GoogleEnabled && cfg.GoogleRedirectURL == "" {
		missing = append(missing, "GOOGLE_OAUTH_REDIRECT_URI")
	}
	if cfg.SessionSecret == "" {
		missing = append(missing, "APP_SESSION_SIGNING_KEY")
	}
	if cfg.AlpacaAPIKeyID == "" {
		missing = append(missing, "ALPACA_API_KEY")
	}
	if cfg.AlpacaAPISecretKey == "" {
		missing = append(missing, "ALPACA_SECRET_KEY")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if len(cfg.SessionSecret) < 32 {
		return Config{}, errors.New("APP_SESSION_SIGNING_KEY must contain at least 32 characters")
	}
	if !validPostgresURL(cfg.DatabaseURL) {
		return Config{}, errors.New("DATABASE_URL must be a PostgreSQL connection string")
	}
	if !validHTTPURL(cfg.FrontendURL) {
		return Config{}, errors.New("FRONTEND_URL must be an HTTP or HTTPS URL")
	}
	if cfg.GoogleEnabled && !validHTTPURL(cfg.GoogleRedirectURL) {
		return Config{}, errors.New("GOOGLE_OAUTH_REDIRECT_URI must be an HTTP or HTTPS URL")
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
	if !validHTTPURL(cfg.NLPAPIURL) {
		return Config{}, errors.New("FINBERT_INFERENCE_URL must be an HTTP or HTTPS URL")
	}
	if !validHTTPURL(cfg.BlueskyAPIURL) {
		return Config{}, errors.New("BLUESKY_API_BASE_URL must be an HTTP or HTTPS URL")
	}
	blueskyEnabled, err := strconv.ParseBool(firstValue("BLUESKY_API_ENABLED"))
	if firstValue("BLUESKY_API_ENABLED") == "" {
		blueskyEnabled = true
		err = nil
	}
	if err != nil {
		return Config{}, errors.New("BLUESKY_API_ENABLED must be true or false")
	}
	cfg.BlueskyEnabled = blueskyEnabled
	if !validAlpacaFeed(cfg.AlpacaDataFeed) {
		return Config{}, fmt.Errorf("unsupported ALPACA_DATA_FEED %q", cfg.AlpacaDataFeed)
	}
	sessionTTLSeconds, err := strconv.Atoi(valueOrDefault("AUTH_SESSION_TTL_SECONDS", "28800"))
	if err != nil || sessionTTLSeconds < 300 || sessionTTLSeconds > 604800 {
		return Config{}, errors.New("AUTH_SESSION_TTL_SECONDS must be between 300 and 604800")
	}
	cfg.SessionTTL = time.Duration(sessionTTLSeconds) * time.Second
	nlpTimeoutSeconds, err := strconv.Atoi(valueOrDefault("NLP_REQUEST_TIMEOUT_SECONDS", "20"))
	if err != nil || nlpTimeoutSeconds < 1 || nlpTimeoutSeconds > 120 {
		return Config{}, errors.New("NLP_REQUEST_TIMEOUT_SECONDS must be between 1 and 120")
	}
	cfg.NLPRequestTimeout = time.Duration(nlpTimeoutSeconds) * time.Second
	databaseTimeoutSeconds, err := strconv.Atoi(valueOrDefault("DATABASE_CONNECT_TIMEOUT_SECONDS", "15"))
	if err != nil || databaseTimeoutSeconds < 1 || databaseTimeoutSeconds > 60 {
		return Config{}, errors.New("DATABASE_CONNECT_TIMEOUT_SECONDS must be between 1 and 60")
	}
	cfg.DatabaseTimeout = time.Duration(databaseTimeoutSeconds) * time.Second
	databaseMinConnections, err := strconv.Atoi(valueOrDefault("DATABASE_MIN_CONNECTIONS", "1"))
	if err != nil || databaseMinConnections < 0 {
		return Config{}, errors.New("DATABASE_MIN_CONNECTIONS must be zero or greater")
	}
	databaseMaxConnections, err := strconv.Atoi(valueOrDefault("DATABASE_MAX_CONNECTIONS", "10"))
	if err != nil || databaseMaxConnections < 1 || databaseMaxConnections < databaseMinConnections {
		return Config{}, errors.New("DATABASE_MAX_CONNECTIONS must be at least DATABASE_MIN_CONNECTIONS")
	}
	cfg.DatabaseMinConns = int32(databaseMinConnections)
	cfg.DatabaseMaxConns = int32(databaseMaxConnections)

	return cfg, nil
}

func validHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func validPostgresURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Host != "" && parsed.User != nil &&
		(parsed.Scheme == "postgres" || parsed.Scheme == "postgresql")
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
