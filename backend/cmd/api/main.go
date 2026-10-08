package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/config"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/market"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/alpaca"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/gdelt"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/googleauth"
	httpapi "github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/transport/http"
)

func main() {
	logger := log.New(os.Stdout, "api ", log.LstdFlags|log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal(err)
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
	authHandler := httpapi.NewGoogleAuthHandler(authClient, cfg.FrontendURL)
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
	tradingHandler := httpapi.NewTradingHandler(tradingClient)
	newsClient, err := gdelt.NewClient(cfg.GDELTAPIURL, market.SP500TopThirty, nil)
	if err != nil {
		logger.Fatal(err)
	}
	newsHandler := httpapi.NewNewsHandler(newsClient)

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
		"/api/v1/trading/portfolio",
		httpapi.RequireAuth(authClient, http.HandlerFunc(tradingHandler.Portfolio)),
	)
	router.Handle(
		"/api/v1/trading/assets",
		httpapi.RequireAuth(authClient, http.HandlerFunc(tradingHandler.Assets)),
	)
	router.Handle(
		"/api/v1/trading/orders",
		httpapi.RequireAuth(authClient, http.HandlerFunc(tradingHandler.SubmitOrder)),
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
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Printf("listening on %s", cfg.Address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal(err)
	}
}
