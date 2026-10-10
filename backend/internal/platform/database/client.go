package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	if err := c.pool.Ping(ctx); err != nil {
		return err
	}

	var missingObjects string
	if err := c.pool.QueryRow(ctx, `
		with required(name, present) as (values
			('public.profiles', to_regclass('public.profiles') is not null),
			('public.user_credentials', to_regclass('public.user_credentials') is not null),
			('public.broker_accounts', to_regclass('public.broker_accounts') is not null),
			('public.account_snapshots', to_regclass('public.account_snapshots') is not null),
			('public.positions', to_regclass('public.positions') is not null),
			('public.position_snapshots', to_regclass('public.position_snapshots') is not null),
			('public.orders', to_regclass('public.orders') is not null),
			('public.fills', to_regclass('public.fills') is not null),
			('public.order_audit_events', to_regclass('public.order_audit_events') is not null),
			('public.sync_trading_state(text,jsonb,jsonb)', to_regprocedure('public.sync_trading_state(text,jsonb,jsonb)') is not null),
			('public.get_trading_state(text,text,text)', to_regprocedure('public.get_trading_state(text,text,text)') is not null)
		)
		select coalesce(string_agg(name, ', ' order by name), '')
		from required
		where not present
	`).Scan(&missingObjects); err != nil {
		return fmt.Errorf("check database migrations: %w", err)
	}
	if missingObjects != "" {
		return fmt.Errorf("database migrations are missing required objects: %s; apply 001_trading_ledger.sql and 002_password_auth.sql", missingObjects)
	}
	return nil
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

func (c *Client) CreateCredentialUser(ctx context.Context, user domain.User, username, passwordHash string) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin credential registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `
		insert into public.profiles (id, email, display_name, avatar_url)
		values ($1, $2, $3, '')
	`, user.ID, user.Email, user.DisplayName); err != nil {
		return credentialWriteError(err)
	}
	if _, err = tx.Exec(ctx, `
		insert into public.user_credentials (user_id, username, email, password_hash)
		values ($1, $2, $3, $4)
	`, user.ID, username, user.Email, passwordHash); err != nil {
		return credentialWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit credential registration: %w", err)
	}
	return nil
}

func (c *Client) CredentialUser(ctx context.Context, identity string) (domain.User, string, error) {
	var user domain.User
	var passwordHash string
	err := c.pool.QueryRow(ctx, `
		select p.id, p.email, p.display_name, p.avatar_url, c.password_hash
		from public.user_credentials c
		join public.profiles p on p.id = c.user_id
		where lower(c.email) = lower($1) or lower(c.username) = lower($1)
		limit 1
	`, identity).Scan(&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", domain.ErrInvalidCredentials
	}
	if err != nil {
		return domain.User{}, "", fmt.Errorf("load credential user: %w", err)
	}
	return user, passwordHash, nil
}

func credentialWriteError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return domain.ErrIdentityTaken
	}
	return fmt.Errorf("store credential user: %w", err)
}

func (c *Client) SyncTradingState(ctx context.Context, user domain.User, portfolio domain.Portfolio, monitor domain.OrderMonitor) error {
	if strings.TrimSpace(user.ID) == "" {
		return errors.New("authenticated user is required to persist trading state")
	}
	if err := c.EnsureUser(ctx, user); err != nil {
		return fmt.Errorf("ensure database profile: %w", err)
	}
	portfolioJSON, monitorJSON, err := encodeTradingState(portfolio, monitor)
	if err != nil {
		return err
	}
	if _, err := c.pool.Exec(ctx, "select public.sync_trading_state($1, $2::jsonb, $3::jsonb)", user.ID, portfolioJSON, monitorJSON); err != nil {
		return fmt.Errorf("sync trading state: %w", err)
	}
	return nil
}

func encodeTradingState(portfolio domain.Portfolio, monitor domain.OrderMonitor) ([]byte, []byte, error) {
	if portfolio.Positions == nil {
		portfolio.Positions = []domain.Position{}
	}
	if monitor.Orders == nil {
		monitor.Orders = []domain.Order{}
	}
	if monitor.Fills == nil {
		monitor.Fills = []domain.Fill{}
	}
	if monitor.AuditTrail == nil {
		monitor.AuditTrail = []domain.OrderAuditEvent{}
	}
	portfolioJSON, err := json.Marshal(portfolio)
	if err != nil {
		return nil, nil, fmt.Errorf("encode portfolio: %w", err)
	}
	monitorJSON, err := json.Marshal(monitor)
	if err != nil {
		return nil, nil, fmt.Errorf("encode order monitor: %w", err)
	}
	return portfolioJSON, monitorJSON, nil
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
