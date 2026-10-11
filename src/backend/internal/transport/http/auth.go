package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	authservice "github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/auth"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

const (
	sessionCookieName  = "trading_wens_session"
	stateCookieName    = "trading_wens_oauth_state"
	verifierCookieName = "trading_wens_oauth_verifier"
)

type GoogleAuthService interface {
	AuthorizationURL(state, challenge string) string
	Exchange(ctx context.Context, code, verifier string) (domain.User, error)
}

type SessionService interface {
	CreateSession(user domain.User) (string, error)
	User(ctx context.Context, token string) (domain.User, error)
	SessionTTL() time.Duration
}

type CredentialAuthService interface {
	SessionService
	Signup(ctx context.Context, username, email, password, displayName string) (domain.User, error)
	Login(ctx context.Context, identity, password string) (domain.User, error)
}

type UserProfileStore interface {
	EnsureUser(ctx context.Context, user domain.User) error
}

type GoogleAuthHandler struct {
	service     GoogleAuthService
	sessions    SessionService
	frontendURL string
	profiles    UserProfileStore
}

func NewGoogleAuthHandler(
	service GoogleAuthService,
	sessions SessionService,
	frontendURL string,
	profileStores ...UserProfileStore,
) *GoogleAuthHandler {
	var profiles UserProfileStore
	if len(profileStores) > 0 {
		profiles = profileStores[0]
	}
	return &GoogleAuthHandler{
		service: service, sessions: sessions, frontendURL: strings.TrimRight(frontendURL, "/"), profiles: profiles,
	}
}

func (h *GoogleAuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}
	state, err := secureToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authentication_failed", "Google sign-in could not be started")
		return
	}
	verifier, err := secureToken(48)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authentication_failed", "Google sign-in could not be started")
		return
	}
	challengeBytes := sha256.Sum256([]byte(verifier))
	setTemporaryCookie(w, r, stateCookieName, state)
	setTemporaryCookie(w, r, verifierCookieName, verifier)
	http.Redirect(w, r, h.service.AuthorizationURL(state, base64.RawURLEncoding.EncodeToString(challengeBytes[:])), http.StatusFound)
}

func (h *GoogleAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}
	clearOAuthCookies(w, r)
	if r.URL.Query().Get("error") != "" {
		h.redirectAuthError(w, r, "Google sign-in was cancelled or denied")
		return
	}
	stateCookie, stateErr := r.Cookie(stateCookieName)
	verifierCookie, verifierErr := r.Cookie(verifierCookieName)
	providedState := r.URL.Query().Get("state")
	if stateErr != nil || verifierErr != nil || providedState == "" || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(providedState)) != 1 {
		h.redirectAuthError(w, r, "The sign-in request expired. Please try again")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		h.redirectAuthError(w, r, "Google did not return an authorization code")
		return
	}
	user, err := h.service.Exchange(r.Context(), code, verifierCookie.Value)
	if err != nil {
		h.redirectAuthError(w, r, "Google sign-in could not be completed")
		return
	}
	if h.profiles != nil {
		if err := h.profiles.EnsureUser(r.Context(), user); err != nil {
			h.redirectAuthError(w, r, "Your application profile could not be prepared")
			return
		}
	}
	session, err := h.sessions.CreateSession(user)
	if err != nil {
		h.redirectAuthError(w, r, "A secure session could not be created")
		return
	}
	setSessionCookie(w, r, session, h.sessions.SessionTTL())
	http.Redirect(w, r, h.frontendURL+"/dashboard", http.StatusFound)
}

func (h *GoogleAuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign-in is required")
		return
	}
	user, err := h.sessions.User(r.Context(), cookie.Value)
	if err != nil {
		clearSessionCookie(w, r)
		writeError(w, http.StatusUnauthorized, "unauthorized", "The session is invalid or expired")
		return
	}
	if h.profiles != nil {
		if err := h.profiles.EnsureUser(r.Context(), user); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database_unavailable", "Your profile is temporarily unavailable")
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]domain.User{"user": user})
}

type CredentialAuthHandler struct {
	service CredentialAuthService
}

func NewCredentialAuthHandler(service CredentialAuthService) *CredentialAuthHandler {
	return &CredentialAuthHandler{service: service}
}

func (h *CredentialAuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}
	var request struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, err := h.service.Signup(r.Context(), request.Username, request.Email, request.Password, request.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrIdentityTaken):
			writeError(w, http.StatusConflict, "identity_taken", err.Error())
		case errors.Is(err, authservice.ErrInvalidSignup):
			writeError(w, http.StatusBadRequest, "invalid_signup", "Use a valid email, a 3–32 character username, and a password of at least 12 characters")
		default:
			writeError(w, http.StatusServiceUnavailable, "authentication_unavailable", "Account creation is temporarily unavailable")
		}
		return
	}
	h.writeSession(w, r, http.StatusCreated, user)
}

func (h *CredentialAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}
	var request struct {
		Identity string `json:"identity"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	user, err := h.service.Login(r.Context(), request.Identity, request.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", domain.ErrInvalidCredentials.Error())
		} else {
			writeError(w, http.StatusServiceUnavailable, "authentication_unavailable", "Sign-in is temporarily unavailable")
		}
		return
	}
	h.writeSession(w, r, http.StatusOK, user)
}

func (h *CredentialAuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign-in is required")
		return
	}
	user, err := h.service.User(r.Context(), cookie.Value)
	if err != nil {
		clearSessionCookie(w, r)
		writeError(w, http.StatusUnauthorized, "unauthorized", "The session is invalid or expired")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]domain.User{"user": user})
}

func (h *CredentialAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}
	clearSessionCookie(w, r)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h *CredentialAuthHandler) writeSession(w http.ResponseWriter, r *http.Request, status int, user domain.User) {
	session, err := h.service.CreateSession(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authentication_failed", "A secure session could not be created")
		return
	}
	setSessionCookie(w, r, session, h.service.SessionTTL())
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, map[string]domain.User{"user": user})
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: value, Path: "/", HttpOnly: true,
		Secure: secureRequest(r), SameSite: http.SameSiteLaxMode,
		MaxAge: int(ttl.Seconds()), Expires: time.Now().Add(ttl),
	})
}

func (h *GoogleAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}
	clearSessionCookie(w, r)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h *GoogleAuthHandler) redirectAuthError(w http.ResponseWriter, r *http.Request, message string) {
	destination := h.frontendURL + "/auth?" + url.Values{"error": {message}}.Encode()
	http.Redirect(w, r, destination, http.StatusFound)
}

func secureToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func setTemporaryCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: "/", HttpOnly: true, Secure: secureRequest(r),
		SameSite: http.SameSiteLaxMode, MaxAge: 600, Expires: time.Now().Add(10 * time.Minute),
	})
}

func clearOAuthCookies(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, r, stateCookieName)
	clearCookie(w, r, verifierCookieName)
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, r, sessionCookieName)
}

func clearCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: "/", HttpOnly: true, Secure: secureRequest(r),
		SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0),
	})
}

func secureRequest(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}
