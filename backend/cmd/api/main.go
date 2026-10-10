package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	authservice "github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/auth"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/config"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/market"
	newsservice "github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/news"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/alpaca"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/bluesky"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/database"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/gdelt"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/googleauth"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/nlp"
	tradingservice "github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/trading"
	httpapi "github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/transport/http"
)

func main() {
	logger := log.New(os.Stdout, "api ", log.LstdFlags|log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal(err)
	}
	startupContext, cancelStartup := context.WithTimeout(context.Background(), cfg.DatabaseTimeout)
	defer cancelStartup()
	databaseClient, err := database.NewClient(startupContext, cfg.DatabaseURL, cfg.DatabaseMinConns, cfg.DatabaseMaxConns)
	if err != nil {
		logger.Fatal(err)
	}
	defer databaseClient.Close()
	if err := databaseClient.Ping(startupContext); err != nil {
		logger.Fatalf("database is unavailable or migrations are missing: %v", err)
	}

	authService, err := authservice.NewService(databaseClient, cfg.SessionSecret, cfg.SessionTTL)
	if err != nil {
		logger.Fatal(err)
	}
	credentialAuthHandler := httpapi.NewCredentialAuthHandler(authService)
	var googleAuthHandler *httpapi.GoogleAuthHandler
	if cfg.GoogleEnabled {
		googleClient, err := googleauth.NewClient(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.GoogleRedirectURL,
			cfg.SessionSecret,
			cfg.SessionTTL,
			nil,
		)
		if err != nil {
			logger.Fatal(err)
		}
		googleAuthHandler = httpapi.NewGoogleAuthHandler(googleClient, authService, cfg.FrontendURL, databaseClient)
	}
	marketClient, err := alpaca.NewClient(
		cfg.AlpacaDataURL,
		cfg.AlpacaAPIKeyID,
		cfg.AlpacaAPISecretKey,
		cfg.AlpacaDataFeed,
		market.SP500TopThirty,
		nil,
	)
	if err != nil {
		logger.Fatal(err)
	}
	marketHandler := httpapi.NewMarketHandler(marketClient)
	tradingClient, err := alpaca.NewTradingClient(
		cfg.AlpacaTradingURL,
		cfg.AlpacaAPIKeyID,
		cfg.AlpacaAPISecretKey,
		nil,
	)
	if err != nil {
		logger.Fatal(err)
	}
	newsClient, err := gdelt.NewClient(cfg.GDELTAPIURL, market.SP500TopThirty, nil)
	if err != nil {
		logger.Fatal(err)
	}
	alpacaNewsClient, err := alpaca.NewNewsClient(
		cfg.AlpacaDataURL,
		cfg.AlpacaAPIKeyID,
		cfg.AlpacaAPISecretKey,
		market.SP500TopThirty,
		nil,
	)
	if err != nil {
		logger.Fatal(err)
	}
	nlpClient, err := nlp.NewClient(cfg.NLPAPIURL, &http.Client{Timeout: cfg.NLPRequestTimeout})
	if err != nil {
		logger.Fatal(err)
	}
	newsService := newsservice.NewService([]newsservice.Provider{newsClient, alpacaNewsClient}, nlpClient)
	newsHandler := httpapi.NewNewsHandler(newsService)
	tradingService := tradingservice.NewService(tradingClient, databaseClient, newsService)
	tradingHandler := httpapi.NewTradingHandler(tradingService)
	var socialService httpapi.SocialService
	if cfg.BlueskyEnabled {
		blueskyClient, err := bluesky.NewClient(cfg.BlueskyAPIURL, market.SP500TopThirty, nil)
		if err != nil {
			logger.Fatal(err)
		}
		socialService = blueskyClient
	}
	socialHandler := httpapi.NewSocialHandler(socialService)

	router := http.NewServeMux()
	router.HandleFunc("/api/v1/auth/providers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Only GET is allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"password":true,"google":%t}`, cfg.GoogleEnabled)))
	})
	router.HandleFunc("/api/v1/auth/google/start", func(w http.ResponseWriter, r *http.Request) {
		if googleAuthHandler == nil {
			http.Error(w, "Google sign-in is not configured", http.StatusServiceUnavailable)
			return
		}
		googleAuthHandler.Start(w, r)
	})
	router.HandleFunc("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		if googleAuthHandler == nil {
			http.Redirect(w, r, cfg.FrontendURL+"/auth?error=Google+sign-in+is+not+configured", http.StatusFound)
			return
		}
		googleAuthHandler.Callback(w, r)
	})
	router.HandleFunc("/api/v1/auth/signup", credentialAuthHandler.Signup)
	router.HandleFunc("/api/v1/auth/login", credentialAuthHandler.Login)
	router.HandleFunc("/api/v1/auth/session", credentialAuthHandler.Session)
	router.HandleFunc("/api/v1/auth/logout", credentialAuthHandler.Logout)
	router.Handle(
		"/api/v1/market/snapshots",
		httpapi.RequireAuth(authService, http.HandlerFunc(marketHandler.Snapshots)),
	)
	router.Handle(
		"/api/v1/market/stocks/{symbol}/history",
		httpapi.RequireAuth(authService, http.HandlerFunc(marketHandler.History)),
	)
	router.Handle(
		"/api/v1/news",
		httpapi.RequireAuth(authService, http.HandlerFunc(newsHandler.Latest)),
	)
	router.Handle(
		"/api/v1/social",
		httpapi.RequireAuth(authService, http.HandlerFunc(socialHandler.Latest)),
	)
	router.Handle(
		"/api/v1/trading/portfolio",
		httpapi.RequireAuth(authService, http.HandlerFunc(tradingHandler.Portfolio)),
	)
	router.Handle(
		"/api/v1/trading/assets",
		httpapi.RequireAuth(authService, http.HandlerFunc(tradingHandler.Assets)),
	)
	router.Handle(
		"/api/v1/trading/orders",
		httpapi.RequireAuth(authService, http.HandlerFunc(tradingHandler.Orders)),
	)
	router.Handle(
		"/api/v1/trading/orders/{orderID}",
		httpapi.RequireAuth(authService, http.HandlerFunc(tradingHandler.OrderActions)),
	)
	router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.CORS(cfg.CORSAllowedOrigins, router),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Printf("listening on %s", cfg.Address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal(err)
	}
}
