package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/supabase"
)

type SessionVerifier interface {
	User(ctx context.Context, accessToken string) (domain.User, error)
}

func RequireAuth(verifier SessionVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		scheme, token, found := strings.Cut(authorization, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "A valid bearer token is required")
			return
		}

		if _, err := verifier.User(r.Context(), strings.TrimSpace(token)); err != nil {
			var apiErr *supabase.APIError
			if errors.As(err, &apiErr) && (apiErr.Status == http.StatusUnauthorized || apiErr.Status == http.StatusForbidden) {
				writeError(w, http.StatusUnauthorized, "unauthorized", "The session is invalid or expired")
				return
			}
			writeError(w, http.StatusBadGateway, "authentication_unavailable", "Authentication service is unavailable")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimRight(strings.TrimSpace(origin), "/")] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
		_, originAllowed := allowed[origin]
		if origin != "" && originAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			if origin == "" || !originAllowed {
				writeError(w, http.StatusForbidden, "origin_not_allowed", "The request origin is not allowed")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
