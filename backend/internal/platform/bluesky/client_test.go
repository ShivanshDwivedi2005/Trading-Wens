package bluesky

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

func TestLatestForSymbolNormalizesFiltersAndCachesPosts(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/xrpc/app.bsky.feed.searchPosts" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Fatalf("public Bluesky search must not send credentials")
		}
		query := r.URL.Query()
		if !strings.Contains(query.Get("q"), `"$AAPL"`) || !strings.Contains(query.Get("q"), `"Apple stock"`) {
			t.Fatalf("unexpected stock query: %q", query.Get("q"))
		}
		if query.Get("limit") != "25" || query.Get("sort") != "latest" || query.Get("lang") != "en" {
			t.Fatalf("unexpected search options: %#v", query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"posts":[
				{
					"uri":"at://did:plc:market/app.bsky.feed.post/3abc",
					"author":{"did":"did:plc:market","handle":"marketdesk.bsky.social","displayName":"Market Desk","avatar":"https://example.com/avatar.jpg"},
					"record":{"text":"$AAPL reports stronger demand","createdAt":"2026-10-08T08:30:00Z","langs":["en"]},
					"indexedAt":"2026-10-08T08:30:01Z","likeCount":14,"replyCount":2,"repostCount":5,"quoteCount":1
				},
				{
					"uri":"at://did:plc:other/app.bsky.feed.post/3other",
					"author":{"did":"did:plc:other","handle":"other.bsky.social"},
					"record":{"text":"Apple pie recipe","createdAt":"2026-10-08T08:29:00Z","langs":["en"]}
				}
			]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, []domain.MarketSymbol{{Symbol: "AAPL", Name: "Apple"}}, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	first, err := client.LatestForSymbol(context.Background(), "aapl")
	if err != nil {
		t.Fatalf("load Bluesky posts: %v", err)
	}
	second, err := client.LatestForSymbol(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("load cached Bluesky posts: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected one provider request, got %d", requests.Load())
	}
	if first.Source != "bluesky" || first.Symbol != "AAPL" || first.Count != 1 || second.Count != 1 {
		t.Fatalf("unexpected feed: %#v", first)
	}
	post := first.Data[0]
	if post.URL != "https://bsky.app/profile/marketdesk.bsky.social/post/3abc" || post.Username != "marketdesk.bsky.social" || post.Metrics.Reposts != 5 {
		t.Fatalf("unexpected post: %#v", post)
	}
}

func TestLatestForSymbolRejectsInvalidTicker(t *testing.T) {
	client, err := NewClient("https://public.api.bsky.app", nil, http.DefaultClient)
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
		http.Error(w, "provider details", http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, nil, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	_, err = client.LatestForSymbol(context.Background(), "AAPL")
	if err == nil || strings.Contains(err.Error(), "provider details") {
		t.Fatalf("expected sanitized provider error, got %v", err)
	}
}
