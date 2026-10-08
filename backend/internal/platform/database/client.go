package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

const maxResponseSize = 2 << 20

type Client struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}

func NewClient(baseURL, serviceKey string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, errors.New("invalid database API URL")
	}
	if strings.TrimSpace(serviceKey) == "" {
		return nil, errors.New("database service key is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    parsed.String(),
		serviceKey: strings.TrimSpace(serviceKey),
		httpClient: httpClient,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.request(ctx, http.MethodGet, "/rest/v1/profiles?select=id&limit=1", nil, "", nil)
}

func (c *Client) EnsureUser(ctx context.Context, user domain.User) error {
	if strings.TrimSpace(user.ID) == "" || strings.TrimSpace(user.Email) == "" {
		return errors.New("authenticated user identity is incomplete")
	}
	payload := []map[string]string{{
		"id":           user.ID,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"avatar_url":   user.AvatarURL,
	}}
	return c.request(
		ctx,
		http.MethodPost,
		"/rest/v1/profiles?on_conflict=id",
		payload,
		"resolution=merge-duplicates,return=minimal",
		nil,
	)
}

func (c *Client) SyncTradingState(
	ctx context.Context,
	user domain.User,
	portfolio domain.Portfolio,
	monitor domain.OrderMonitor,
) error {
	if strings.TrimSpace(user.ID) == "" {
		return errors.New("authenticated user is required to persist trading state")
	}
	if err := c.EnsureUser(ctx, user); err != nil {
		return fmt.Errorf("ensure database profile: %w", err)
	}
	payload := map[string]any{
		"p_user_id":   user.ID,
		"p_portfolio": portfolio,
		"p_monitor":   monitor,
	}
	return c.request(ctx, http.MethodPost, "/rest/v1/rpc/sync_trading_state", payload, "return=minimal", nil)
}

type tradingState struct {
	Portfolio domain.Portfolio    `json:"portfolio"`
	Monitor   domain.OrderMonitor `json:"monitor"`
}

func (c *Client) TradingState(
	ctx context.Context,
	user domain.User,
	provider, accountID string,
) (domain.Portfolio, domain.OrderMonitor, error) {
	payload := map[string]string{
		"p_user_id": user.ID, "p_provider": provider, "p_account_id": accountID,
	}
	var state tradingState
	if err := c.request(
		ctx,
		http.MethodPost,
		"/rest/v1/rpc/get_trading_state",
		payload,
		"",
		&state,
	); err != nil {
		return domain.Portfolio{}, domain.OrderMonitor{}, err
	}
	return state.Portfolio, state.Monitor, nil
}

func (c *Client) request(
	ctx context.Context,
	method, path string,
	payload any,
	prefer string,
	destination any,
) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode database request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create database request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send database request: %w", err)
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("read database response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(res.StatusCode, responseBody)
	}
	if destination != nil && len(bytes.TrimSpace(responseBody)) > 0 {
		if err := json.Unmarshal(responseBody, destination); err != nil {
			return fmt.Errorf("decode database response: %w", err)
		}
	}
	return nil
}

func decodeAPIError(status int, body []byte) error {
	var response struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details string `json:"details"`
	}
	_ = json.Unmarshal(body, &response)
	message := strings.TrimSpace(response.Message)
	if message == "" {
		message = http.StatusText(status)
	}
	if response.Details != "" {
		message += ": " + response.Details
	}
	return &APIError{Status: status, Code: response.Code, Message: message}
}
