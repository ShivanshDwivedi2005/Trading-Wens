package supabase

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginCreatesProfile(t *testing.T) {
	t.Helper()

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("apikey") != "publishable-key" {
			t.Fatalf("expected publishable key header")
		}

		switch r.URL.Path {
		case "/auth/v1/token":
			if r.URL.Query().Get("grant_type") != "password" {
				t.Fatalf("expected password grant")
			}
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode login request: %v", err)
			}
			if request["email"] != "analyst@example.com" || request["password"] != "strong-password" {
				t.Fatalf("unexpected login request: %#v", request)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"access_token":"access-token",
				"refresh_token":"refresh-token",
				"expires_in":3600,
				"token_type":"bearer",
				"user":{
					"id":"user-1",
					"email":"analyst@example.com",
					"email_confirmed_at":"2026-10-05T08:30:00Z",
					"user_metadata":{"full_name":"Risk Analyst"}
				}
			}`))
		case "/rest/v1/profiles":
			if r.Header.Get("Authorization") != "Bearer access-token" {
				t.Fatalf("expected user access token")
			}
			if r.Header.Get("Prefer") != "resolution=merge-duplicates,return=minimal" {
				t.Fatalf("expected profile upsert preference")
			}
			if r.URL.Query().Get("on_conflict") != "id" {
				t.Fatalf("expected id conflict target")
			}
			var profile map[string]string
			if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
				t.Fatalf("decode profile request: %v", err)
			}
			if profile["id"] != "user-1" || profile["display_name"] != "Risk Analyst" {
				t.Fatalf("unexpected profile: %#v", profile)
			}
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "publishable-key", server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	result, err := client.Login(context.Background(), "analyst@example.com", "strong-password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected two provider requests, got %d", requests)
	}
	if result.User.ID != "user-1" || result.User.DisplayName != "Risk Analyst" {
		t.Fatalf("unexpected user: %#v", result.User)
	}
	if result.Session == nil || result.Session.AccessToken != "access-token" {
		t.Fatalf("unexpected session: %#v", result.Session)
	}
}

func TestLoginReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_code":"invalid_credentials","msg":"Invalid login credentials"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "publishable-key", server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = client.Login(context.Background(), "analyst@example.com", "wrong-password")
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Status != http.StatusBadRequest || apiErr.Code != "invalid_credentials" {
		t.Fatalf("unexpected API error: %#v", apiErr)
	}
}

func TestSignupRequiresEmailConfirmation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/signup" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("apikey") != "publishable-key" {
			t.Fatalf("expected publishable key header")
		}

		var request struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Data     struct {
				FullName string `json:"full_name"`
			} `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode signup request: %v", err)
		}
		if request.Email != "new@example.com" || request.Password != "strong-password" {
			t.Fatalf("unexpected signup credentials")
		}
		if request.Data.FullName != "New Analyst" {
			t.Fatalf("unexpected full name: %q", request.Data.FullName)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"user":{
				"id":"user-2",
				"email":"new@example.com",
				"user_metadata":{"full_name":"New Analyst"}
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "publishable-key", server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	result, err := client.Signup(context.Background(), "new@example.com", "strong-password", "New Analyst")
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if result.User.ID != "user-2" || result.User.DisplayName != "New Analyst" {
		t.Fatalf("unexpected user: %#v", result.User)
	}
	if result.Session != nil {
		t.Fatalf("expected no session before email confirmation")
	}
	if !result.EmailConfirmationRequired {
		t.Fatalf("expected email confirmation to be required")
	}
}

func TestSignupCreatesProfileForImmediateSession(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/auth/v1/signup":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"access_token":"access-token",
				"refresh_token":"refresh-token",
				"expires_in":3600,
				"token_type":"bearer",
				"user":{
					"id":"user-3",
					"email":"ready@example.com",
					"user_metadata":{"full_name":"Ready Analyst"}
				}
			}`))
		case "/rest/v1/profiles":
			if r.Header.Get("Authorization") != "Bearer access-token" {
				t.Fatalf("expected user access token")
			}
			w.WriteHeader(http.StatusCreated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "publishable-key", server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	result, err := client.Signup(context.Background(), "ready@example.com", "strong-password", "Ready Analyst")
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected signup and profile requests, got %d", requests)
	}
	if result.Session == nil || result.Session.AccessToken != "access-token" {
		t.Fatalf("unexpected session: %#v", result.Session)
	}
	if result.EmailConfirmationRequired {
		t.Fatalf("did not expect email confirmation requirement")
	}
}

func TestUserValidatesAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/user" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("apikey") != "publishable-key" || r.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatal("expected Supabase authentication headers")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"user-1",
			"email":"analyst@example.com",
			"email_confirmed_at":"2026-10-06T10:00:00Z",
			"user_metadata":{"full_name":"Risk Analyst"}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "publishable-key", server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	user, err := client.User(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.ID != "user-1" || user.DisplayName != "Risk Analyst" {
		t.Fatalf("unexpected user: %#v", user)
	}
}
