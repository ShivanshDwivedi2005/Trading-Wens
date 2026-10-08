package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type marketServiceStub struct {
	snapshots func(context.Context) (domain.MarketSnapshotSet, error)
	history   func(context.Context, string, string) (domain.StockHistory, error)
}

func (s marketServiceStub) History(ctx context.Context, symbol, historyRange string) (domain.StockHistory, error) {
	return s.history(ctx, symbol, historyRange)
}

func (s marketServiceStub) Snapshots(ctx context.Context) (domain.MarketSnapshotSet, error) {
	return s.snapshots(ctx)
}

type sessionVerifierStub struct {
	user func(context.Context, string) (domain.User, error)
}

func (s sessionVerifierStub) User(ctx context.Context, token string) (domain.User, error) {
	return s.user(ctx, token)
}

func TestMarketSnapshotsHandler(t *testing.T) {
	handler := NewMarketHandler(marketServiceStub{snapshots: func(context.Context) (domain.MarketSnapshotSet, error) {
		return domain.MarketSnapshotSet{
			Data:   []domain.MarketSnapshot{{Symbol: "AAPL", Price: 251.5, Available: true}},
			AsOf:   time.Date(2026, 10, 6, 16, 0, 0, 0, time.UTC),
			Feed:   "iex",
			Source: "alpaca",
			Count:  1,
		}, nil
	}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/snapshots", nil)
	res := httptest.NewRecorder()
	handler.Snapshots(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if res.Header().Get("Cache-Control") != "private, max-age=10" {
		t.Fatalf("unexpected cache control: %s", res.Header().Get("Cache-Control"))
	}
}

func TestRequireAuth(t *testing.T) {
	verifier := sessionVerifierStub{user: func(_ context.Context, token string) (domain.User, error) {
		if token != "valid-token" {
			return domain.User{}, errors.New("invalid session")
		}
		return domain.User{ID: "user-1"}, nil
	}}
	protected := RequireAuth(verifier, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := domain.AuthenticatedUser(r.Context())
		if !ok || user.ID != "user-1" {
			t.Fatalf("authenticated user was not added to request context: %#v", user)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	unauthorized := httptest.NewRecorder()
	protected.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing token to return 401, got %d", unauthorized.Code)
	}

	authorizedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	authorizedRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "valid-token"})
	authorized := httptest.NewRecorder()
	protected.ServeHTTP(authorized, authorizedRequest)
	if authorized.Code != http.StatusNoContent {
		t.Fatalf("expected valid token to pass, got %d", authorized.Code)
	}
}

func TestMarketSnapshotsHandlerHidesProviderErrors(t *testing.T) {
	handler := NewMarketHandler(marketServiceStub{snapshots: func(context.Context) (domain.MarketSnapshotSet, error) {
		return domain.MarketSnapshotSet{}, errors.New("provider details")
	}})
	res := httptest.NewRecorder()
	handler.Snapshots(res, httptest.NewRequest(http.MethodGet, "/api/v1/market/snapshots", nil))
	if res.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", res.Code)
	}
}

func TestMarketHistoryHandler(t *testing.T) {
	handler := NewMarketHandler(marketServiceStub{
		snapshots: func(context.Context) (domain.MarketSnapshotSet, error) { return domain.MarketSnapshotSet{}, nil },
		history: func(_ context.Context, symbol, historyRange string) (domain.StockHistory, error) {
			if symbol != "AAPL" || historyRange != "5D" {
				t.Fatalf("unexpected history request: %s %s", symbol, historyRange)
			}
			return domain.StockHistory{Symbol: symbol, Range: historyRange, Source: "alpaca", Count: 1}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/stocks/AAPL/history?range=5D", nil)
	req.SetPathValue("symbol", "AAPL")
	res := httptest.NewRecorder()
	handler.History(res, req)
	if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "private, max-age=30" {
		t.Fatalf("unexpected response: %d %#v", res.Code, res.Header())
	}
}

func TestMarketHistoryHandlerRejectsInvalidRange(t *testing.T) {
	handler := NewMarketHandler(marketServiceStub{
		snapshots: func(context.Context) (domain.MarketSnapshotSet, error) { return domain.MarketSnapshotSet{}, nil },
		history: func(context.Context, string, string) (domain.StockHistory, error) {
			return domain.StockHistory{}, domain.ErrUnsupportedRange
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/stocks/AAPL/history?range=1Y", nil)
	req.SetPathValue("symbol", "AAPL")
	res := httptest.NewRecorder()
	handler.History(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	handler := CORS([]string{"http://localhost:3000"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/market/snapshots", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent || res.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("unexpected CORS response: %d %#v", res.Code, res.Header())
	}
	if res.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("expected credentialed CORS response")
	}
}
