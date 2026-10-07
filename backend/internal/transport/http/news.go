package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type NewsService interface {
	Latest(ctx context.Context) (domain.NewsFeed, error)
	LatestForSymbol(ctx context.Context, symbol string) (domain.NewsFeed, error)
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

	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	var (
		result domain.NewsFeed
		err    error
	)
	if symbol == "" {
		result, err = h.service.Latest(r.Context())
	} else {
		result, err = h.service.LatestForSymbol(r.Context(), symbol)
	}
	if errors.Is(err, domain.ErrUnsupportedSymbol) {
		writeError(w, http.StatusNotFound, "symbol_not_found", "This stock is not in the supported market universe")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "news_unavailable", "Market news is temporarily unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=120")
	writeJSON(w, http.StatusOK, result)
}
