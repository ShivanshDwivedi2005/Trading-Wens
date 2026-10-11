package database

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

func TestNewClientRejectsInvalidDatabaseURL(t *testing.T) {
	if _, err := NewClient(context.Background(), "not-a-database-url", 0, 4); err == nil {
		t.Fatal("expected invalid database URL error")
	}
}

func TestNewClientConfiguresPool(t *testing.T) {
	client, err := NewClient(context.Background(), "postgresql://user:password@localhost:5432/trading?sslmode=disable", 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.pool.Config().MinConns != 1 || client.pool.Config().MaxConns != 4 {
		t.Fatalf("unexpected pool limits: %d/%d", client.pool.Config().MinConns, client.pool.Config().MaxConns)
	}
}

func TestEncodeTradingStateUsesEmptyArraysForPortfolioOnlySync(t *testing.T) {
	portfolioJSON, monitorJSON, err := encodeTradingState(domain.Portfolio{}, domain.OrderMonitor{})
	if err != nil {
		t.Fatal(err)
	}
	var portfolio map[string]any
	if err := json.Unmarshal(portfolioJSON, &portfolio); err != nil {
		t.Fatal(err)
	}
	if positions, ok := portfolio["positions"].([]any); !ok || len(positions) != 0 {
		t.Fatalf("expected an empty positions array, got %#v", portfolio["positions"])
	}
	var monitor map[string]any
	if err := json.Unmarshal(monitorJSON, &monitor); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"orders", "fills", "audit_trail"} {
		items, ok := monitor[field].([]any)
		if !ok || len(items) != 0 {
			t.Fatalf("expected an empty %s array, got %#v", field, monitor[field])
		}
	}
}
