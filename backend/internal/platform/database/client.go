package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	pool *pgxpool.Pool
}

func NewClient(ctx context.Context, databaseURL string, minConnections, maxConnections int32) (*Client, error) {
	config, err := pgxpool.ParseConfig(strings.TrimSpace(databaseURL))
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	config.MinConns = minConnections
	config.MaxConns = maxConnections
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return &Client{pool: pool}, nil
}

func (c *Client) Close() {
	c.pool.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *Client) EnsureUser(ctx context.Context, user domain.User) error {
	if strings.TrimSpace(user.ID) == "" || strings.TrimSpace(user.Email) == "" {
		return errors.New("authenticated user identity is incomplete")
	}
	_, err := c.pool.Exec(ctx, `
		insert into public.profiles (id, email, display_name, avatar_url, updated_at)
		values ($1, $2, $3, $4, now())
		on conflict (id) do update set
			email = excluded.email,
			display_name = excluded.display_name,
			avatar_url = excluded.avatar_url,
			updated_at = now()
	`, user.ID, user.Email, user.DisplayName, user.AvatarURL)
	if err != nil {
		return fmt.Errorf("upsert database profile: %w", err)
	}
	return nil
}

func (c *Client) SyncTradingState(ctx context.Context, user domain.User, portfolio domain.Portfolio, monitor domain.OrderMonitor) error {
	if strings.TrimSpace(user.ID) == "" {
		return errors.New("authenticated user is required to persist trading state")
	}
	if err := c.EnsureUser(ctx, user); err != nil {
		return fmt.Errorf("ensure database profile: %w", err)
	}
	portfolioJSON, err := json.Marshal(portfolio)
	if err != nil {
		return fmt.Errorf("encode portfolio: %w", err)
	}
	monitorJSON, err := json.Marshal(monitor)
	if err != nil {
		return fmt.Errorf("encode order monitor: %w", err)
	}
	if _, err := c.pool.Exec(ctx, "select public.sync_trading_state($1, $2::jsonb, $3::jsonb)", user.ID, portfolioJSON, monitorJSON); err != nil {
		return fmt.Errorf("sync trading state: %w", err)
	}
	return nil
}

type tradingState struct {
	Portfolio domain.Portfolio    `json:"portfolio"`
	Monitor   domain.OrderMonitor `json:"monitor"`
}

func (c *Client) TradingState(ctx context.Context, user domain.User, provider, accountID string) (domain.Portfolio, domain.OrderMonitor, error) {
	var encoded []byte
	if err := c.pool.QueryRow(ctx, "select public.get_trading_state($1, $2, $3)", user.ID, provider, accountID).Scan(&encoded); err != nil {
		return domain.Portfolio{}, domain.OrderMonitor{}, fmt.Errorf("load trading state: %w", err)
	}
	var state tradingState
	if err := json.Unmarshal(encoded, &state); err != nil {
		return domain.Portfolio{}, domain.OrderMonitor{}, fmt.Errorf("decode trading state: %w", err)
	}
	return state.Portfolio, state.Monitor, nil
}
