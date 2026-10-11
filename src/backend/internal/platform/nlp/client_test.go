package nlp

import (
	"context"
	"encoding/json"
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

func TestAnalyzeSplitsLargeRequestsIntoBoundedBatches(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var request inferenceRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Texts) > analysisBatchSize {
			t.Fatalf("batch exceeded limit: %d", len(request.Texts))
		}
		predictions := make([]prediction, len(request.Texts))
		for index := range predictions {
			predictions[index] = prediction{
				Label: "NEUTRAL", Confidence: 0.8, SentimentScore: 0,
				Uncertainty: 0.2, Probabilities: map[string]float64{"NEGATIVE": 0.1, "NEUTRAL": 0.8, "POSITIVE": 0.1},
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": map[string]string{"name": "test", "version": "one"}, "predictions": predictions,
		})
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	texts := make([]string, analysisBatchSize+1)
	for index := range texts {
		texts[index] = "headline"
	}
	results, err := client.Analyze(context.Background(), texts)
	if err != nil || len(results) != len(texts) || requestCount != 2 {
		t.Fatalf("unexpected batching result: count=%d requests=%d error=%v", len(results), requestCount, err)
	}
}
