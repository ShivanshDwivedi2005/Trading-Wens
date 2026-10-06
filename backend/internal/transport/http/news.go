package httpapi

import (
	"context"
	"net/http"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type NewsService interface {
	Latest(ctx context.Context) (domain.NewsFeed, error)
}

type NewsHandler struct {
	service NewsService
}

func NewNewsHandler(service NewsService) *NewsHandler {
	return &NewsHandler{service: service}
}

func (h *NewsHandler) Latest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}

	result, err := h.service.Latest(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "news_unavailable", "Market news is temporarily unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=120")
	writeJSON(w, http.StatusOK, result)
}
