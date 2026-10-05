package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type authServiceStub struct {
	login func(context.Context, string, string) (domain.AuthResult, error)
}

func (s authServiceStub) Login(ctx context.Context, email, password string) (domain.AuthResult, error) {
	return s.login(ctx, email, password)
}

func TestLoginHandler(t *testing.T) {
	service := authServiceStub{login: func(_ context.Context, email, password string) (domain.AuthResult, error) {
		if email != "analyst@example.com" || password != "strong-password" {
			t.Fatalf("unexpected credentials: %q %q", email, password)
		}
		return domain.AuthResult{
			User: domain.User{ID: "user-1", Email: email},
			Session: &domain.Session{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    3600,
				TokenType:    "bearer",
			},
		}, nil
	}}
	handler := NewAuthHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(
		`{"email":" Analyst@Example.com ","password":"strong-password"}`,
	))
	res := httptest.NewRecorder()
	handler.Login(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store response")
	}

	var response domain.AuthResult
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Session == nil || response.Session.AccessToken != "access-token" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestLoginHandlerRejectsInvalidInput(t *testing.T) {
	service := authServiceStub{login: func(context.Context, string, string) (domain.AuthResult, error) {
		t.Fatal("service should not be called")
		return domain.AuthResult{}, nil
	}}
	handler := NewAuthHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(
		`{"email":"not-an-email","password":"short"}`,
	))
	res := httptest.NewRecorder()
	handler.Login(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", res.Code)
	}
}

func TestLoginHandlerHidesInternalErrors(t *testing.T) {
	service := authServiceStub{login: func(context.Context, string, string) (domain.AuthResult, error) {
		return domain.AuthResult{}, errors.New("connection details")
	}}
	handler := NewAuthHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(
		`{"email":"analyst@example.com","password":"strong-password"}`,
	))
	res := httptest.NewRecorder()
	handler.Login(res, req)

	if res.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", res.Code)
	}
	if strings.Contains(res.Body.String(), "connection details") {
		t.Fatalf("response leaked internal error: %s", res.Body.String())
	}
}
