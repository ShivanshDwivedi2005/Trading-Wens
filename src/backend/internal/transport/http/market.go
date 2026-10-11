package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type MarketService interface {
	Snapshots(ctx context.Context) (domain.MarketSnapshotSet, error)
	History(ctx context.Context, symbol, historyRange string) (domain.StockHistory, error)
}

func (h *MarketHandler) History(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
	result, err := h.service.History(r.Context(), symbol, r.URL.Query().Get("range"))
	if errors.Is(err, domain.ErrUnsupportedSymbol) {
		writeError(w, http.StatusNotFound, "symbol_not_found", "This stock is not in the supported market universe")
		return
	}
	if errors.Is(err, domain.ErrUnsupportedRange) {
		writeError(w, http.StatusBadRequest, "invalid_range", "Range must be one of 1D, 5D, or 1M")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "market_data_unavailable", "Live market history is temporarily unavailable")
		return
	}

	w.Header().Set("Cache-Control", "private, max-age=30")
	writeJSON(w, http.StatusOK, result)
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
