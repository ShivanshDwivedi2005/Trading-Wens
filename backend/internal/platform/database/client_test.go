package database

import (
	"context"
	"testing"
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
