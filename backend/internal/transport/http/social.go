package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type SocialService interface {
	LatestForSymbol(ctx context.Context, symbol string) (domain.SocialFeed, error)
}

type SocialHandler struct {
	service SocialService
}

func NewSocialHandler(service SocialService) *SocialHandler {
	return &SocialHandler{service: service}
}

func (h *SocialHandler) Latest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "bluesky_not_configured", "Bluesky market conversation is not configured")
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol_required", "A stock symbol is required")
		return
	}
	result, err := h.service.LatestForSymbol(r.Context(), symbol)
	if errors.Is(err, domain.ErrUnsupportedSymbol) {
		writeError(w, http.StatusNotFound, "symbol_not_found", "The stock symbol is invalid")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "social_unavailable", "Bluesky market conversation is temporarily unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=120")
	writeJSON(w, http.StatusOK, result)
}
