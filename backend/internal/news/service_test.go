package news

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type providerStub struct {
	feed domain.NewsFeed
	err  error
}

func (s providerStub) Latest(context.Context) (domain.NewsFeed, error) {
	return s.feed, s.err
}

func (s providerStub) LatestForSymbol(context.Context, string) (domain.NewsFeed, error) {
	return s.feed, s.err
}

type analyzerStub struct{}

func (analyzerStub) Analyze(_ context.Context, texts []string) ([]domain.SentimentAnalysis, error) {
	results := make([]domain.SentimentAnalysis, len(texts))
	for index := range results {
		results[index] = domain.SentimentAnalysis{Label: "POSITIVE", Score: 0.72, Confidence: 0.88}
	}
	return results, nil
}

func TestServiceAggregatesDeduplicatesAndAnalyzes(t *testing.T) {
	older := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	providers := []Provider{
		providerStub{feed: domain.NewsFeed{Source: "gdelt", Data: []domain.NewsArticle{
			{ID: "one", Title: "Older", URL: "https://example.com/one", PublishedAt: older},
		}}},
		providerStub{feed: domain.NewsFeed{Source: "alpaca", Data: []domain.NewsArticle{
			{ID: "duplicate", Title: "Duplicate", URL: "https://example.com/one", PublishedAt: older},
			{ID: "two", Title: "Newer", URL: "https://example.com/two", PublishedAt: newer},
		}}},
	}
	service := NewService(providers, analyzerStub{})
	feed, err := service.Latest(context.Background())
	if err != nil {
		t.Fatalf("collect news: %v", err)
	}
	if feed.Count != 2 || feed.Data[0].ID != "two" || feed.Data[0].Sentiment == nil {
		t.Fatalf("unexpected feed: %#v", feed)
	}
	if len(feed.Providers) != 2 || feed.Providers[0] != "alpaca" || feed.Providers[1] != "gdelt" {
		t.Fatalf("unexpected providers: %#v", feed.Providers)
	}
}

func TestServiceUsesHealthyProviderWhenAnotherFails(t *testing.T) {
	service := NewService([]Provider{
		providerStub{err: errors.New("provider failed")},
		providerStub{feed: domain.NewsFeed{Source: "gdelt", Data: []domain.NewsArticle{{ID: "one", Title: "Available"}}}},
	}, nil)
	feed, err := service.Latest(context.Background())
	if err != nil || feed.Count != 1 {
		t.Fatalf("expected degraded feed, got %#v, %v", feed, err)
	}
}
