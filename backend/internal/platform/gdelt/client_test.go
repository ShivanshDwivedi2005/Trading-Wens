package gdelt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestNormalizesAndCachesArticles(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Query().Get("mode") != "artlist" || r.URL.Query().Get("format") != "json" || r.URL.Query().Get("sort") != "datedesc" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"articles":[
			{"url":"https://example.com/nvidia-news","title":"NVIDIA expands data center platform","seendate":"20261006T155959Z","socialimage":"https://example.com/image.jpg","domain":"example.com","language":"English","sourcecountry":"United States"},
			{"url":"https://example.com/nvidia-news","title":"Duplicate","seendate":"20261006T155959Z","domain":"example.com","language":"English","sourcecountry":"United States"}
		]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
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

	if requests != 1 || first.Count != 1 || len(second.Data) != 1 {
		t.Fatalf("unexpected cache result: requests=%d feed=%#v", requests, first)
	}
	if first.Data[0].MatchedSymbols[0] != "NVDA" || first.Data[0].ID == "" {
		t.Fatalf("unexpected article: %#v", first.Data[0])
	}
}

func TestLatestRejectsProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := client.Latest(context.Background()); err == nil {
		t.Fatal("expected provider error")
	}
}
