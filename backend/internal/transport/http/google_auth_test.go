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

type googleAuthStub struct {
	exchange func(context.Context, string, string) (domain.User, error)
}

type profileStoreStub struct {
	user domain.User
}

func (s *profileStoreStub) EnsureUser(_ context.Context, user domain.User) error {
	s.user = user
	return nil
}

func (s googleAuthStub) AuthorizationURL(state, challenge string) string {
	return "https://accounts.google.com/auth?state=" + state + "&challenge=" + challenge
}

func (s googleAuthStub) Exchange(ctx context.Context, code, verifier string) (domain.User, error) {
	return s.exchange(ctx, code, verifier)
}

func (s googleAuthStub) CreateSession(user domain.User) (string, error) {
	return "signed-session-for-" + user.ID, nil
}

func (s googleAuthStub) User(_ context.Context, token string) (domain.User, error) {
	if token != "signed-session-for-user-1" {
		return domain.User{}, context.Canceled
	}
	return domain.User{ID: "user-1", Email: "analyst@example.com", DisplayName: "Analyst"}, nil
}

func (s googleAuthStub) SessionTTL() time.Duration { return 8 * time.Hour }

func TestGoogleAuthStartSetsStateAndPKCECookies(t *testing.T) {
	handler := NewGoogleAuthHandler(googleAuthStub{}, "http://localhost:3000")
	res := httptest.NewRecorder()
	handler.Start(res, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/start", nil))
	if res.Code != http.StatusFound || !strings.HasPrefix(res.Header().Get("Location"), "https://accounts.google.com/") {
		t.Fatalf("unexpected response: %d %s", res.Code, res.Header().Get("Location"))
	}
	if len(res.Result().Cookies()) != 2 {
		t.Fatalf("expected state and verifier cookies, got %d", len(res.Result().Cookies()))
	}
}

func TestGoogleAuthCallbackCreatesSession(t *testing.T) {
	service := googleAuthStub{exchange: func(_ context.Context, code, verifier string) (domain.User, error) {
		if code != "google-code" || verifier != "pkce-verifier" {
			t.Fatalf("unexpected exchange values: %q %q", code, verifier)
		}
		return domain.User{ID: "user-1", Email: "analyst@example.com"}, nil
	}}
	profiles := &profileStoreStub{}
	handler := NewGoogleAuthHandler(service, "http://localhost:3000", profiles)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=google-code&state=expected-state", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "expected-state"})
	req.AddCookie(&http.Cookie{Name: verifierCookieName, Value: "pkce-verifier"})
	res := httptest.NewRecorder()
	handler.Callback(res, req)
	if res.Code != http.StatusFound || res.Header().Get("Location") != "http://localhost:3000/dashboard" {
		t.Fatalf("unexpected callback response: %d %s", res.Code, res.Header().Get("Location"))
	}
	var sessionFound bool
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == sessionCookieName && cookie.Value == "signed-session-for-user-1" && cookie.HttpOnly {
			sessionFound = true
		}
	}
	if !sessionFound {
		t.Fatal("expected secure application session cookie")
	}
	if profiles.user.ID != "user-1" {
		t.Fatalf("profile was not provisioned: %#v", profiles.user)
	}
}

func TestGoogleAuthCallbackRejectsStateMismatch(t *testing.T) {
	handler := NewGoogleAuthHandler(googleAuthStub{exchange: func(context.Context, string, string) (domain.User, error) {
		t.Fatal("exchange must not run")
		return domain.User{}, nil
	}}, "http://localhost:3000")
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=code&state=wrong", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "expected"})
	req.AddCookie(&http.Cookie{Name: verifierCookieName, Value: "verifier"})
	res := httptest.NewRecorder()
	handler.Callback(res, req)
	if res.Code != http.StatusFound || !strings.Contains(res.Header().Get("Location"), "/auth?error=") {
		t.Fatalf("unexpected rejection: %d %s", res.Code, res.Header().Get("Location"))
	}
}

func TestGoogleSessionAndLogout(t *testing.T) {
	handler := NewGoogleAuthHandler(googleAuthStub{}, "http://localhost:3000")
	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	sessionReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "signed-session-for-user-1"})
	sessionRes := httptest.NewRecorder()
	handler.Session(sessionRes, sessionReq)
	if sessionRes.Code != http.StatusOK || !strings.Contains(sessionRes.Body.String(), "analyst@example.com") {
		t.Fatalf("unexpected session response: %d %s", sessionRes.Code, sessionRes.Body.String())
	}

	logoutRes := httptest.NewRecorder()
	handler.Logout(logoutRes, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil))
	if logoutRes.Code != http.StatusNoContent || len(logoutRes.Result().Cookies()) != 1 || logoutRes.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("expected logout to clear the session cookie")
	}
}
