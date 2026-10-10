package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type credentialAuthStub struct {
	signup func(string, string, string, string) (domain.User, error)
	login  func(string, string) (domain.User, error)
}

func (s credentialAuthStub) Signup(_ context.Context, username, email, password, displayName string) (domain.User, error) {
	return s.signup(username, email, password, displayName)
}

func (s credentialAuthStub) Login(_ context.Context, identity, password string) (domain.User, error) {
	return s.login(identity, password)
}

func (credentialAuthStub) CreateSession(domain.User) (string, error) { return "signed-session", nil }
func (credentialAuthStub) User(context.Context, string) (domain.User, error) {
	return domain.User{ID: "usr_1", Email: "analyst@example.com"}, nil
}
func (credentialAuthStub) SessionTTL() time.Duration { return time.Hour }

func TestCredentialSignupCreatesSession(t *testing.T) {
	service := credentialAuthStub{signup: func(username, email, password, displayName string) (domain.User, error) {
		if username != "analyst" || email != "analyst@example.com" || password != "a-secure-password" || displayName != "Analyst" {
			t.Fatalf("unexpected signup fields: %q %q %q %q", username, email, password, displayName)
		}
		return domain.User{ID: "usr_1", Email: email, DisplayName: displayName}, nil
	}}
	handler := NewCredentialAuthHandler(service)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", strings.NewReader(`{"username":"analyst","email":"analyst@example.com","password":"a-secure-password","display_name":"Analyst"}`))
	response := httptest.NewRecorder()
	handler.Signup(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
	if cookies := response.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != sessionCookieName || !cookies[0].HttpOnly {
		t.Fatal("expected HTTP-only session cookie")
	}
}

func TestCredentialLoginUsesGenericFailure(t *testing.T) {
	service := credentialAuthStub{login: func(string, string) (domain.User, error) {
		return domain.User{}, domain.ErrInvalidCredentials
	}}
	handler := NewCredentialAuthHandler(service)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"identity":"missing","password":"wrong"}`))
	response := httptest.NewRecorder()
	handler.Login(response, request)
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), "invalid email, username, or password") {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}
