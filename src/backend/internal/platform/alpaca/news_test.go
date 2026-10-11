package alpaca

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

func TestNewsClientNormalizesAndCachesArticles(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v1beta1/news" || r.URL.Query().Get("symbols") != "AAPL,MSFT" || r.URL.Query().Get("sort") != "desc" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		if r.Header.Get("APCA-API-KEY-ID") != "key" || r.Header.Get("APCA-API-SECRET-KEY") != "secret" {
			t.Fatal("missing Alpaca credentials")
		}
		_, _ = w.Write([]byte(`{"news":[{
			"id":123,"headline":"Apple expands services revenue","summary":"Quarterly services revenue rose.",
			"url":"https://example.com/apple","source":"benzinga","created_at":"2026-10-08T12:30:00Z",
			"symbols":["AAPL"],"images":[{"size":"large","url":"https://example.com/apple.jpg"}]
		}]}`))
	}))
	defer server.Close()

	client, err := NewNewsClient(server.URL, "key", "secret", []domain.MarketSymbol{
		{Symbol: "AAPL", Name: "Apple"}, {Symbol: "MSFT", Name: "Microsoft"},
	}, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	first, err := client.Latest(context.Background())
	if err != nil {
		t.Fatalf("fetch news: %v", err)
	}
	second, err := client.Latest(context.Background())
	if err != nil {
		t.Fatalf("fetch cached news: %v", err)
	}
	if requests != 1 || first.Count != 1 || second.Data[0].Provider != "alpaca" {
		t.Fatalf("unexpected result: requests=%d feed=%#v", requests, first)
	}
	if first.Data[0].Summary == "" || first.Data[0].MatchedSymbols[0] != "AAPL" {
		t.Fatalf("unexpected article: %#v", first.Data[0])
	}
}

func TestNewsClientBuildsFocusedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("symbols") != "NVDA" || r.URL.Query().Get("limit") != "25" {
			t.Fatalf("unexpected request: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"news":[]}`))
	}))
	defer server.Close()
	client, err := NewNewsClient(server.URL, "key", "secret", []domain.MarketSymbol{{Symbol: "AAPL"}}, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := client.LatestForSymbol(context.Background(), " nvda "); err != nil {
		t.Fatalf("fetch focused news: %v", err)
	}
}

func TestNewsClientRejectsInvalidSymbol(t *testing.T) {
	client, err := NewNewsClient("https://data.alpaca.markets", "key", "secret", []domain.MarketSymbol{{Symbol: "AAPL"}}, nil)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := client.LatestForSymbol(context.Background(), strings.Repeat("A", 16)); err == nil {
		t.Fatal("expected invalid symbol error")
	}
}
