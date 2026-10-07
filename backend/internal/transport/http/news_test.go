package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type newsServiceStub struct {
	latest          func(context.Context) (domain.NewsFeed, error)
	latestForSymbol func(context.Context, string) (domain.NewsFeed, error)
}

func (s newsServiceStub) LatestForSymbol(ctx context.Context, symbol string) (domain.NewsFeed, error) {
	return s.latestForSymbol(ctx, symbol)
}

func (s newsServiceStub) Latest(ctx context.Context) (domain.NewsFeed, error) {
	return s.latest(ctx)
}

func TestNewsHandler(t *testing.T) {
	handler := NewNewsHandler(newsServiceStub{latest: func(context.Context) (domain.NewsFeed, error) {
		return domain.NewsFeed{Data: []domain.NewsArticle{{ID: "one", Title: "Headline"}}, Source: "gdelt", Count: 1}, nil
	}})
	res := httptest.NewRecorder()
	handler.Latest(res, httptest.NewRequest(http.MethodGet, "/api/v1/news", nil))
	if res.Code != http.StatusOK || res.Header().Get("Cache-Control") != "private, max-age=120" {
		t.Fatalf("unexpected response: %d %#v", res.Code, res.Header())
	}
}

func TestNewsHandlerHidesProviderFailure(t *testing.T) {
	handler := NewNewsHandler(newsServiceStub{latest: func(context.Context) (domain.NewsFeed, error) {
		return domain.NewsFeed{}, errors.New("provider details")
	}})
	res := httptest.NewRecorder()
	handler.Latest(res, httptest.NewRequest(http.MethodGet, "/api/v1/news", nil))
	if res.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", res.Code)
	}
}

func TestNewsHandlerFiltersBySymbol(t *testing.T) {
	handler := NewNewsHandler(newsServiceStub{
		latest: func(context.Context) (domain.NewsFeed, error) { return domain.NewsFeed{}, nil },
		latestForSymbol: func(_ context.Context, symbol string) (domain.NewsFeed, error) {
			if symbol != "AAPL" {
				t.Fatalf("unexpected symbol: %s", symbol)
			}
			return domain.NewsFeed{Data: []domain.NewsArticle{{ID: "one", MatchedSymbols: []string{symbol}}}, Source: "gdelt", Count: 1}, nil
		},
	})
	res := httptest.NewRecorder()
	handler.Latest(res, httptest.NewRequest(http.MethodGet, "/api/v1/news?symbol=aapl", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
}
