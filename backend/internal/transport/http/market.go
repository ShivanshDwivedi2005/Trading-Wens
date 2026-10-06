package httpapi

import (
	"context"
	"net/http"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type MarketService interface {
	Snapshots(ctx context.Context) (domain.MarketSnapshotSet, error)
}

type MarketHandler struct {
	service MarketService
}

func NewMarketHandler(service MarketService) *MarketHandler {
	return &MarketHandler{service: service}
}

func (h *MarketHandler) Snapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}

	result, err := h.service.Snapshots(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "market_data_unavailable", "Live market data is temporarily unavailable")
		return
	}

	w.Header().Set("Cache-Control", "private, max-age=10")
	writeJSON(w, http.StatusOK, result)
}
