package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

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

	authClient, err := googleauth.NewClient(
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
	authHandler := httpapi.NewGoogleAuthHandler(authClient, cfg.FrontendURL, databaseClient)
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
	router.HandleFunc("/api/v1/auth/google/start", authHandler.Start)
	router.HandleFunc("/auth/google/callback", authHandler.Callback)
	router.HandleFunc("/api/v1/auth/session", authHandler.Session)
	router.HandleFunc("/api/v1/auth/logout", authHandler.Logout)
	router.Handle(
		"/api/v1/market/snapshots",
		httpapi.RequireAuth(authClient, http.HandlerFunc(marketHandler.Snapshots)),
	)
	router.Handle(
		"/api/v1/market/stocks/{symbol}/history",
		httpapi.RequireAuth(authClient, http.HandlerFunc(marketHandler.History)),
	)
	router.Handle(
		"/api/v1/news",
		httpapi.RequireAuth(authClient, http.HandlerFunc(newsHandler.Latest)),
	)
	router.Handle(
		"/api/v1/social",
		httpapi.RequireAuth(authClient, http.HandlerFunc(socialHandler.Latest)),
	)
	router.Handle(
		"/api/v1/trading/portfolio",
		httpapi.RequireAuth(authClient, http.HandlerFunc(tradingHandler.Portfolio)),
	)
	router.Handle(
		"/api/v1/trading/assets",
		httpapi.RequireAuth(authClient, http.HandlerFunc(tradingHandler.Assets)),
	)
	router.Handle(
		"/api/v1/trading/orders",
		httpapi.RequireAuth(authClient, http.HandlerFunc(tradingHandler.Orders)),
	)
	router.Handle(
		"/api/v1/trading/orders/{orderID}",
		httpapi.RequireAuth(authClient, http.HandlerFunc(tradingHandler.OrderActions)),
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
