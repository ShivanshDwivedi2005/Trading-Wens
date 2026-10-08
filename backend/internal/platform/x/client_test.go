package x

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

func TestLatestForSymbolNormalizesAndCachesPosts(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/2/tweets/search/recent" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}
		query := r.URL.Query()
		if !strings.Contains(query.Get("query"), "$AAPL") || !strings.Contains(query.Get("query"), `"Apple"`) {
			t.Fatalf("unexpected stock query: %q", query.Get("query"))
		}
		if query.Get("max_results") != "10" || query.Get("sort_order") != "recency" {
			t.Fatalf("unexpected search options: %#v", query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":[{
				"id":"199",
				"text":"Apple reports stronger demand",
				"author_id":"7",
				"created_at":"2026-10-08T08:30:00Z",
				"lang":"en",
				"public_metrics":{"like_count":14,"reply_count":2,"repost_count":5,"quote_count":1}
			}],
			"includes":{"users":[{"id":"7","name":"Market Desk","username":"marketdesk","profile_image_url":"https://example.com/avatar.jpg"}]}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-token", []domain.MarketSymbol{{Symbol: "AAPL", Name: "Apple"}}, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	first, err := client.LatestForSymbol(context.Background(), "aapl")
	if err != nil {
		t.Fatalf("load X posts: %v", err)
	}
	second, err := client.LatestForSymbol(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("load cached X posts: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected one provider request, got %d", requests.Load())
	}
	if first.Source != "x" || first.Symbol != "AAPL" || first.Count != 1 || second.Count != 1 {
		t.Fatalf("unexpected feed: %#v", first)
	}
	post := first.Data[0]
	if post.URL != "https://x.com/marketdesk/status/199" || post.Username != "marketdesk" || post.Metrics.Reposts != 5 {
		t.Fatalf("unexpected post: %#v", post)
	}
}

func TestLatestForSymbolRejectsInvalidTicker(t *testing.T) {
	client, err := NewClient("https://api.x.com", "test-token", nil, http.DefaultClient)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	_, err = client.LatestForSymbol(context.Background(), "not a ticker")
	if err != domain.ErrUnsupportedSymbol {
		t.Fatalf("expected unsupported symbol, got %v", err)
	}
}

func TestLatestForSymbolHidesProviderBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "secret provider details", http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "test-token", nil, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	_, err = client.LatestForSymbol(context.Background(), "AAPL")
	if err == nil || strings.Contains(err.Error(), "secret provider details") {
		t.Fatalf("expected sanitized provider error, got %v", err)
	}
}
