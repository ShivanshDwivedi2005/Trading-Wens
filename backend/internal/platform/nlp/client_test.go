package nlp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnalyzeReturnsCalibratedPrediction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sentiment" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":{"name":"trading-wens-sentiment","version":"2026-10-08"},
			"predictions":[{
				"label":"NEGATIVE","confidence":0.91,"sentiment_score":-0.82,
				"uncertainty":0.24,"probabilities":{"NEGATIVE":0.91,"NEUTRAL":0.08,"POSITIVE":0.01}
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	results, err := client.Analyze(context.Background(), []string{"Company cuts its outlook"})
	if err != nil {
		t.Fatalf("analyze text: %v", err)
	}
	if len(results) != 1 || results[0].Label != "NEGATIVE" || results[0].Score != -0.82 || results[0].ModelVersion != "2026-10-08" {
		t.Fatalf("unexpected analysis: %#v", results)
	}
}

func TestAnalyzeRejectsMismatchedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":{"name":"test","version":"one"},"predictions":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := client.Analyze(context.Background(), []string{"headline"}); err == nil {
		t.Fatal("expected prediction count error")
	}
}
