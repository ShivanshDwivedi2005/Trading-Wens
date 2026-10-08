package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type socialServiceStub struct {
	latestForSymbol func(context.Context, string) (domain.SocialFeed, error)
}

func (s socialServiceStub) LatestForSymbol(ctx context.Context, symbol string) (domain.SocialFeed, error) {
	return s.latestForSymbol(ctx, symbol)
}

func TestSocialHandler(t *testing.T) {
	handler := NewSocialHandler(socialServiceStub{latestForSymbol: func(_ context.Context, symbol string) (domain.SocialFeed, error) {
		if symbol != "AAPL" {
			t.Fatalf("unexpected symbol: %s", symbol)
		}
		return domain.SocialFeed{Data: []domain.SocialPost{{ID: "one"}}, Source: "x", Symbol: symbol, Count: 1}, nil
	}})
	res := httptest.NewRecorder()
	handler.Latest(res, httptest.NewRequest(http.MethodGet, "/api/v1/social?symbol=aapl", nil))
	if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "private, max-age=120" {
		t.Fatalf("unexpected response: %d %#v", res.Code, res.Header())
	}
}

func TestSocialHandlerRequiresSymbol(t *testing.T) {
	handler := NewSocialHandler(socialServiceStub{latestForSymbol: func(context.Context, string) (domain.SocialFeed, error) {
		return domain.SocialFeed{}, nil
	}})
	res := httptest.NewRecorder()
	handler.Latest(res, httptest.NewRequest(http.MethodGet, "/api/v1/social", nil))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}
}

func TestSocialHandlerReportsDisabledProvider(t *testing.T) {
	res := httptest.NewRecorder()
	NewSocialHandler(nil).Latest(res, httptest.NewRequest(http.MethodGet, "/api/v1/social?symbol=AAPL", nil))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}
}

func TestSocialHandlerHidesProviderFailure(t *testing.T) {
	handler := NewSocialHandler(socialServiceStub{latestForSymbol: func(context.Context, string) (domain.SocialFeed, error) {
		return domain.SocialFeed{}, errors.New("provider details")
	}})
	res := httptest.NewRecorder()
	handler.Latest(res, httptest.NewRequest(http.MethodGet, "/api/v1/social?symbol=AAPL", nil))
	if res.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", res.Code)
	}
}
