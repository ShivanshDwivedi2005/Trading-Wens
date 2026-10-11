package googleauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

func TestAuthorizationURLUsesOIDCAndPKCE(t *testing.T) {
	client, err := NewClient("client-id", "client-secret", "http://localhost:8080/auth/google/callback", "01234567890123456789012345678901", time.Hour, nil)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	parsed, err := url.Parse(client.AuthorizationURL("state", "challenge"))
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	query := parsed.Query()
	if query.Get("scope") != "openid email profile" || query.Get("state") != "state" || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("unexpected authorization query: %s", parsed.RawQuery)
	}
}

func TestExchangeReturnsVerifiedGoogleUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil || r.Form.Get("code_verifier") != "verifier" || r.Form.Get("client_secret") != "client-secret" {
				t.Fatalf("unexpected token request: %#v", r.Form)
			}
			_, _ = w.Write([]byte(`{"access_token":"google-access","token_type":"Bearer"}`))
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer google-access" {
				t.Fatal("missing Google access token")
			}
			_, _ = w.Write([]byte(`{"sub":"google-user-1","email":"analyst@example.com","email_verified":true,"name":"Market Analyst","picture":"https://example.com/avatar.png"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient("client-id", "client-secret", "http://localhost:8080/auth/google/callback", "01234567890123456789012345678901", time.Hour, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	client.tokenURL = server.URL + "/token"
	client.userInfoURL = server.URL + "/userinfo"
	user, err := client.Exchange(context.Background(), "code", "verifier")
	if err != nil {
		t.Fatalf("exchange code: %v", err)
	}
	if user.ID != "google-user-1" || user.Email != "analyst@example.com" || user.DisplayName != "Market Analyst" {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestSessionRejectsTampering(t *testing.T) {
	client, err := NewClient("client-id", "client-secret", "http://localhost:8080/auth/google/callback", "01234567890123456789012345678901", time.Hour, nil)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	token, err := client.CreateSession(domain.User{ID: "user-1", Email: "analyst@example.com"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := client.User(context.Background(), token); err != nil {
		t.Fatalf("verify session: %v", err)
	}
	if _, err := client.User(context.Background(), token+"tampered"); err != ErrInvalidSession {
		t.Fatalf("expected invalid session, got %v", err)
	}
}
