package supabase

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

const maxResponseSize = 1 << 20

type Client struct {
	baseURL    string
	apiKey     string
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

func NewClient(baseURL, apiKey string, httpClient *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid Supabase URL")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("Supabase publishable key is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		baseURL:    parsedURL.String(),
		apiKey:     apiKey,
		httpClient: httpClient,
	}, nil
}

func (c *Client) Login(ctx context.Context, email, password string) (domain.AuthResult, error) {
	payload := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{Email: email, Password: password}

	result, err := c.authenticate(ctx, "/auth/v1/token?grant_type=password", payload)
	if err != nil {
		return domain.AuthResult{}, err
	}
	if result.Session == nil || result.Session.AccessToken == "" {
		return domain.AuthResult{}, errors.New("Supabase login response did not contain a session")
	}

	if err := c.ensureProfile(ctx, result.User, result.Session.AccessToken); err != nil {
		return domain.AuthResult{}, fmt.Errorf("create user profile: %w", err)
	}

	return result, nil
}

func (c *Client) authenticate(ctx context.Context, path string, payload any) (domain.AuthResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("encode auth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apikey", c.apiKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("send auth request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("read auth response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return domain.AuthResult{}, decodeAPIError(res.StatusCode, responseBody)
	}

	var providerResponse providerAuthResponse
	if err := json.Unmarshal(responseBody, &providerResponse); err != nil {
		return domain.AuthResult{}, fmt.Errorf("decode auth response: %w", err)
	}
	return providerResponse.normalize(), nil
}

func (c *Client) ensureProfile(ctx context.Context, user domain.User, accessToken string) error {
	payload := struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name,omitempty"`
	}{ID: user.ID, DisplayName: user.DisplayName}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode profile: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/rest/v1/profiles?on_conflict=id",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create profile request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Prefer", "resolution=merge-duplicates,return=minimal")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send profile request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
		if readErr != nil {
			return fmt.Errorf("read profile response: %w", readErr)
		}
		return decodeAPIError(res.StatusCode, responseBody)
	}

	return nil
}

type providerAuthResponse struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	ExpiresIn    int              `json:"expires_in"`
	TokenType    string           `json:"token_type"`
	User         providerUser     `json:"user"`
	Session      *providerSession `json:"session"`
}

type providerSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type providerUser struct {
	ID               string         `json:"id"`
	Email            string         `json:"email"`
	EmailConfirmedAt string         `json:"email_confirmed_at"`
	UserMetadata     map[string]any `json:"user_metadata"`
}

func (r providerAuthResponse) normalize() domain.AuthResult {
	result := domain.AuthResult{
		User: domain.User{
			ID:               r.User.ID,
			Email:            r.User.Email,
			EmailConfirmedAt: r.User.EmailConfirmedAt,
		},
	}
	if displayName, ok := r.User.UserMetadata["full_name"].(string); ok {
		result.User.DisplayName = strings.TrimSpace(displayName)
	}

	if r.Session != nil && r.Session.AccessToken != "" {
		result.Session = &domain.Session{
			AccessToken:  r.Session.AccessToken,
			RefreshToken: r.Session.RefreshToken,
			ExpiresIn:    r.Session.ExpiresIn,
			TokenType:    r.Session.TokenType,
		}
	} else if r.AccessToken != "" {
		result.Session = &domain.Session{
			AccessToken:  r.AccessToken,
			RefreshToken: r.RefreshToken,
			ExpiresIn:    r.ExpiresIn,
			TokenType:    r.TokenType,
		}
	}

	return result
}

func decodeAPIError(status int, body []byte) error {
	var response struct {
		Code             string `json:"code"`
		ErrorCode        string `json:"error_code"`
		Message          string `json:"message"`
		Msg              string `json:"msg"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &response)

	code := response.ErrorCode
	if code == "" {
		code = response.Code
	}
	if code == "" {
		code = "authentication_failed"
	}

	message := response.Message
	if message == "" {
		message = response.Msg
	}
	if message == "" {
		message = response.ErrorDescription
	}
	if message == "" {
		message = "Authentication request failed"
	}

	return &APIError{Status: status, Code: code, Message: message}
}
