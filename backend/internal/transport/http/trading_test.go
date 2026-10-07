package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type tradingServiceStub struct {
	portfolio func(context.Context) (domain.Portfolio, error)
	assets    func(context.Context, string) ([]domain.TradingAsset, error)
	order     func(context.Context, domain.OrderRequest) (domain.Order, error)
}

func (s tradingServiceStub) Portfolio(ctx context.Context) (domain.Portfolio, error) {
	return s.portfolio(ctx)
}

func (s tradingServiceStub) Assets(ctx context.Context, search string) ([]domain.TradingAsset, error) {
	return s.assets(ctx, search)
}

func (s tradingServiceStub) SubmitOrder(ctx context.Context, request domain.OrderRequest) (domain.Order, error) {
	return s.order(ctx, request)
}

func TestTradingHandlerPortfolioAndAssets(t *testing.T) {
	handler := NewTradingHandler(tradingServiceStub{
		portfolio: func(context.Context) (domain.Portfolio, error) {
			return domain.Portfolio{Mode: "paper", Source: "alpaca"}, nil
		},
		assets: func(_ context.Context, search string) ([]domain.TradingAsset, error) {
			if search != "apple" {
				t.Fatalf("unexpected search: %s", search)
			}
			return []domain.TradingAsset{{Symbol: "AAPL", Tradable: true}}, nil
		},
		order: func(context.Context, domain.OrderRequest) (domain.Order, error) { return domain.Order{}, nil },
	})
	portfolioResponse := httptest.NewRecorder()
	handler.Portfolio(portfolioResponse, httptest.NewRequest(http.MethodGet, "/api/v1/trading/portfolio", nil))
	if portfolioResponse.Code != http.StatusOK {
		t.Fatalf("portfolio status: %d", portfolioResponse.Code)
	}
	assetResponse := httptest.NewRecorder()
	handler.Assets(assetResponse, httptest.NewRequest(http.MethodGet, "/api/v1/trading/assets?search=apple", nil))
	if assetResponse.Code != http.StatusOK {
		t.Fatalf("assets status: %d", assetResponse.Code)
	}
}

func TestTradingHandlerValidatesAndSubmitsPaperOrder(t *testing.T) {
	handler := NewTradingHandler(tradingServiceStub{
		portfolio: func(context.Context) (domain.Portfolio, error) { return domain.Portfolio{}, nil },
		assets:    func(context.Context, string) ([]domain.TradingAsset, error) { return nil, nil },
		order: func(_ context.Context, request domain.OrderRequest) (domain.Order, error) {
			if request.Symbol != "AAPL" || request.Quantity != 2 || request.Type != "market" {
				t.Fatalf("unexpected request: %#v", request)
			}
			return domain.Order{ID: "order-1", Mode: "paper"}, nil
		},
	})
	body := bytes.NewBufferString(`{"symbol":"aapl","quantity":2,"side":"buy","type":"market","time_in_force":"day"}`)
	response := httptest.NewRecorder()
	handler.SubmitOrder(response, httptest.NewRequest(http.MethodPost, "/api/v1/trading/orders", body))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}

	invalid := httptest.NewRecorder()
	handler.SubmitOrder(invalid, httptest.NewRequest(http.MethodPost, "/api/v1/trading/orders", bytes.NewBufferString(`{"symbol":"AAPL","quantity":0,"side":"buy","type":"market","time_in_force":"day"}`)))
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", invalid.Code)
	}
}
