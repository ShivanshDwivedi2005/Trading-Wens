package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/config"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/supabase"
	httpapi "github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/transport/http"
)

func main() {
	logger := log.New(os.Stdout, "api ", log.LstdFlags|log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal(err)
	}

	authClient, err := supabase.NewClient(cfg.SupabaseURL, cfg.SupabasePublishableKey, nil)
	if err != nil {
		logger.Fatal(err)
	}
	authHandler := httpapi.NewAuthHandler(authClient)

	router := http.NewServeMux()
	router.HandleFunc("/api/v1/auth/login", authHandler.Login)
	router.HandleFunc("/api/v1/auth/signup", authHandler.Signup)
	router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           router,
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
