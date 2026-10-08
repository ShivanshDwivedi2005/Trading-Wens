package googleauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

var ErrInvalidSession = errors.New("invalid or expired session")

type Client struct {
	clientID     string
	clientSecret string
	redirectURL  string
	sessionKey   []byte
	sessionTTL   time.Duration
	httpClient   *http.Client
	authURL      string
	tokenURL     string
	userInfoURL  string
}

func NewClient(clientID, clientSecret, redirectURL, sessionSecret string, sessionTTL time.Duration, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return nil, errors.New("Google OAuth credentials are required")
	}
	if parsed, err := url.Parse(strings.TrimSpace(redirectURL)); err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("invalid Google redirect URL")
	}
	if len(sessionSecret) < 32 {
		return nil, errors.New("session secret must contain at least 32 characters")
	}
	if sessionTTL <= 0 {
		return nil, errors.New("session TTL must be positive")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		clientID: strings.TrimSpace(clientID), clientSecret: strings.TrimSpace(clientSecret),
		redirectURL: strings.TrimSpace(redirectURL), sessionKey: []byte(sessionSecret),
		sessionTTL: sessionTTL, httpClient: httpClient,
		authURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		tokenURL:    "https://oauth2.googleapis.com/token",
		userInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
	}, nil
}

func (c *Client) AuthorizationURL(state, challenge string) string {
	query := url.Values{
		"client_id":             {c.clientID},
		"redirect_uri":          {c.redirectURL},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"prompt":                {"select_account"},
	}
	return c.authURL + "?" + query.Encode()
}

func (c *Client) Exchange(ctx context.Context, code, verifier string) (domain.User, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"redirect_uri":  {c.redirectURL},
		"grant_type":    {"authorization_code"},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return domain.User{}, fmt.Errorf("create Google token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return domain.User{}, fmt.Errorf("exchange Google authorization code: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return domain.User{}, fmt.Errorf("read Google token response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return domain.User{}, errors.New("Google rejected the authorization code")
	}
	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &token); err != nil || token.AccessToken == "" || !strings.EqualFold(token.TokenType, "Bearer") {
		return domain.User{}, errors.New("Google returned an invalid token response")
	}

	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoURL, nil)
	if err != nil {
		return domain.User{}, fmt.Errorf("create Google user request: %w", err)
	}
	userReq.Header.Set("Accept", "application/json")
	userReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	userRes, err := c.httpClient.Do(userReq)
	if err != nil {
		return domain.User{}, fmt.Errorf("fetch Google user: %w", err)
	}
	defer userRes.Body.Close()
	userBody, err := io.ReadAll(io.LimitReader(userRes.Body, maxResponseSize))
	if err != nil {
		return domain.User{}, fmt.Errorf("read Google user response: %w", err)
	}
	if userRes.StatusCode != http.StatusOK {
		return domain.User{}, errors.New("Google user profile is unavailable")
	}
	var profile struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.Unmarshal(userBody, &profile); err != nil {
		return domain.User{}, errors.New("Google returned an invalid user profile")
	}
	if profile.Subject == "" || profile.Email == "" || !profile.EmailVerified {
		return domain.User{}, errors.New("a verified Google account is required")
	}
	return domain.User{ID: profile.Subject, Email: profile.Email, DisplayName: profile.Name, AvatarURL: profile.Picture}, nil
}

type sessionClaims struct {
	User      domain.User `json:"user"`
	IssuedAt  int64       `json:"iat"`
	ExpiresAt int64       `json:"exp"`
}

func (c *Client) CreateSession(user domain.User) (string, error) {
	now := time.Now().UTC()
	payload, err := json.Marshal(sessionClaims{User: user, IssuedAt: now.Unix(), ExpiresAt: now.Add(c.sessionTTL).Unix()})
	if err != nil {
		return "", fmt.Errorf("encode session: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + c.signature(encoded), nil
}

func (c *Client) User(_ context.Context, token string) (domain.User, error) {
	encoded, providedSignature, found := strings.Cut(token, ".")
	if !found || !hmac.Equal([]byte(providedSignature), []byte(c.signature(encoded))) {
		return domain.User{}, ErrInvalidSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return domain.User{}, ErrInvalidSession
	}
	var claims sessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil || claims.User.ID == "" || claims.User.Email == "" || time.Now().Unix() >= claims.ExpiresAt {
		return domain.User{}, ErrInvalidSession
	}
	return claims.User, nil
}

func (c *Client) SessionTTL() time.Duration {
	return c.sessionTTL
}

func (c *Client) signature(payload string) string {
	digest := hmac.New(sha256.New, c.sessionKey)
	_, _ = digest.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(digest.Sum(nil))
}
