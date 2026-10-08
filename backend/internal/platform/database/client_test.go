package database

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

func TestEnsureUserAndSyncTradingState(t *testing.T) {
	requests := make([]string, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") != "service-key" || r.Header.Get("Authorization") != "Bearer service-key" {
			t.Fatal("database service credentials were not sent")
		}
		requests = append(requests, r.URL.Path)
		var payload any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if r.URL.Path == "/rest/v1/rpc/sync_trading_state" {
			values, ok := payload.(map[string]any)
			if !ok || values["p_user_id"] != "google-user-1" {
				t.Fatalf("unexpected tenant payload: %#v", payload)
			}
		}
		if r.URL.Path == "/rest/v1/rpc/get_trading_state" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"portfolio":{"account":{"id":"paper-account-1"},"positions":[],"as_of":"2026-10-09T00:00:00Z","source":"alpaca","mode":"paper"},"monitor":{"orders":[],"fills":[],"audit_trail":[],"as_of":"2026-10-09T00:00:00Z","source":"alpaca","mode":"paper","order_count":0,"working_count":0,"fill_count":0}}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "service-key", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	user := domain.User{ID: "google-user-1", Email: "analyst@example.com", DisplayName: "Analyst"}
	if err := client.EnsureUser(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	portfolio := domain.Portfolio{
		Account: domain.TradingAccount{ID: "paper-account-1", Status: "ACTIVE"},
		AsOf:    time.Now().UTC(), Source: "alpaca", Mode: "paper",
	}
	if err := client.SyncTradingState(context.Background(), user, portfolio, domain.OrderMonitor{}); err != nil {
		t.Fatal(err)
	}
	storedPortfolio, _, err := client.TradingState(context.Background(), user, "alpaca", "paper-account-1")
	if err != nil || storedPortfolio.Account.ID != "paper-account-1" {
		t.Fatalf("load trading state: %#v %v", storedPortfolio, err)
	}
	if len(requests) != 4 || requests[0] != "/rest/v1/profiles" || requests[1] != "/rest/v1/profiles" || requests[2] != "/rest/v1/rpc/sync_trading_state" || requests[3] != "/rest/v1/rpc/get_trading_state" {
		t.Fatalf("unexpected requests: %#v", requests)
	}
}

func TestNewClientRejectsMissingServiceKey(t *testing.T) {
	if _, err := NewClient("https://example.supabase.co", "", nil); err == nil {
		t.Fatal("expected missing key error")
	}
}
