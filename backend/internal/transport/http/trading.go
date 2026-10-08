package httpapi

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/alpaca"
)

var tradingSymbolPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,14}$`)

type TradingService interface {
	Portfolio(ctx context.Context) (domain.Portfolio, error)
	Assets(ctx context.Context, search string) ([]domain.TradingAsset, error)
	SubmitOrder(ctx context.Context, request domain.OrderRequest) (domain.Order, error)
	OrderMonitor(ctx context.Context) (domain.OrderMonitor, error)
	CancelOrder(ctx context.Context, orderID string) error
}

func (h *TradingHandler) Orders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.OrderMonitor(w, r)
	case http.MethodPost:
		h.SubmitOrder(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET and POST are allowed")
	}
}

func (h *TradingHandler) OrderMonitor(w http.ResponseWriter, r *http.Request) {
	monitor, err := h.service.OrderMonitor(r.Context())
	if err != nil {
		writeTradingError(w, err, "Order activity is temporarily unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, no-cache")
	writeJSON(w, http.StatusOK, monitor)
}

func (h *TradingHandler) OrderActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only DELETE is allowed")
		return
	}
	orderID := strings.TrimSpace(r.PathValue("orderID"))
	if !validOrderID(orderID) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_order_id", "Enter a valid order ID")
		return
	}
	if err := h.service.CancelOrder(r.Context(), orderID); err != nil {
		writeTradingError(w, err, "Paper order could not be canceled")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

type TradingHandler struct {
	service TradingService
}

func NewTradingHandler(service TradingService) *TradingHandler {
	return &TradingHandler{service: service}
}

func (h *TradingHandler) Portfolio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}
	portfolio, err := h.service.Portfolio(r.Context())
	if err != nil {
		writeTradingError(w, err, "Portfolio data is temporarily unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=5")
	writeJSON(w, http.StatusOK, portfolio)
}

func (h *TradingHandler) Assets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed")
		return
	}
	assets, err := h.service.Assets(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		writeTradingError(w, err, "Tradable assets are temporarily unavailable")
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=60")
	writeJSON(w, http.StatusOK, map[string]any{
		"data": assets, "count": len(assets), "source": "alpaca", "mode": "paper",
	})
}

func (h *TradingHandler) SubmitOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}
	var request domain.OrderRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.Side = strings.ToLower(strings.TrimSpace(request.Side))
	request.Type = strings.ToLower(strings.TrimSpace(request.Type))
	request.TimeInForce = strings.ToLower(strings.TrimSpace(request.TimeInForce))
	if !tradingSymbolPattern.MatchString(request.Symbol) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_symbol", "Enter a valid stock symbol")
		return
	}
	if request.Quantity <= 0 || request.Quantity > 1_000_000 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_quantity", "Quantity must be greater than zero")
		return
	}
	if request.Side != "buy" && request.Side != "sell" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_side", "Side must be buy or sell")
		return
	}
	if request.Type != "market" && request.Type != "limit" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_type", "Order type must be market or limit")
		return
	}
	if request.TimeInForce != "day" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_time_in_force", "Paper orders currently use day time in force")
		return
	}
	if request.Type == "limit" && request.LimitPrice <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_limit_price", "Enter a limit price greater than zero")
		return
	}

	order, err := h.service.SubmitOrder(r.Context(), request)
	if err != nil {
		writeTradingError(w, err, "Paper order could not be submitted")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, order)
}

func writeTradingError(w http.ResponseWriter, err error, fallback string) {
	var apiErr *alpaca.TradingAPIError
	if errors.As(err, &apiErr) {
		status := apiErr.Status
		if status < http.StatusBadRequest || status >= http.StatusInternalServerError {
			status = http.StatusBadGateway
		}
		writeError(w, status, "alpaca_trading_error", apiErr.Message)
		return
	}
	writeError(w, http.StatusBadGateway, "trading_unavailable", fallback)
}

func validOrderID(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '-' {
			continue
		}
		return false
	}
	return true
}
