package news

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

const maxAggregatedArticles = 50

type Provider interface {
	Latest(ctx context.Context) (domain.NewsFeed, error)
	LatestForSymbol(ctx context.Context, symbol string) (domain.NewsFeed, error)
}

type Analyzer interface {
	Analyze(ctx context.Context, texts []string) ([]domain.SentimentAnalysis, error)
}

type Service struct {
	providers []Provider
	analyzer  Analyzer
}

func NewService(providers []Provider, analyzer Analyzer) *Service {
	return &Service{providers: append([]Provider(nil), providers...), analyzer: analyzer}
}

func (s *Service) Latest(ctx context.Context) (domain.NewsFeed, error) {
	return s.collect(ctx, "")
}

func (s *Service) LatestForSymbol(ctx context.Context, symbol string) (domain.NewsFeed, error) {
	return s.collect(ctx, strings.ToUpper(strings.TrimSpace(symbol)))
}

func (s *Service) collect(ctx context.Context, symbol string) (domain.NewsFeed, error) {
	type result struct {
		feed domain.NewsFeed
		err  error
	}
	results := make(chan result, len(s.providers))
	for _, provider := range s.providers {
		provider := provider
		go func() {
			var feed domain.NewsFeed
			var err error
			if symbol == "" {
				feed, err = provider.Latest(ctx)
			} else {
				feed, err = provider.LatestForSymbol(ctx, symbol)
			}
			results <- result{feed: feed, err: err}
		}()
	}

	articles := make([]domain.NewsArticle, 0, 50)
	providers := make([]string, 0, len(s.providers))
	seen := make(map[string]struct{})
	var firstError error
	for range s.providers {
		result := <-results
		if result.err != nil {
			if firstError == nil {
				firstError = result.err
			}
			continue
		}
		providers = append(providers, result.feed.Providers...)
		if len(result.feed.Providers) == 0 && result.feed.Source != "" {
			providers = append(providers, result.feed.Source)
		}
		for _, article := range result.feed.Data {
			key := canonicalArticleKey(article)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			articles = append(articles, article)
		}
	}
	if len(articles) == 0 && firstError != nil {
		return domain.NewsFeed{}, firstError
	}

	sort.SliceStable(articles, func(i, j int) bool {
		return articles[i].PublishedAt.After(articles[j].PublishedAt)
	})
	if len(articles) > maxAggregatedArticles {
		articles = articles[:maxAggregatedArticles]
	}
	if s.analyzer != nil && len(articles) > 0 {
		texts := make([]string, len(articles))
		for index, article := range articles {
			texts[index] = strings.TrimSpace(article.Title + ". " + article.Summary)
		}
		if analyses, err := s.analyzer.Analyze(ctx, texts); err == nil && len(analyses) == len(articles) {
			for index := range articles {
				analysis := analyses[index]
				articles[index].Sentiment = &analysis
			}
		}
	}

	providers = uniqueSorted(providers)
	return domain.NewsFeed{
		Data:      articles,
		AsOf:      time.Now().UTC(),
		Source:    "aggregated",
		Providers: providers,
		Count:     len(articles),
	}, nil
}

func canonicalArticleKey(article domain.NewsArticle) string {
	if value := strings.ToLower(strings.TrimSpace(article.URL)); value != "" {
		return value
	}
	return strings.ToLower(strings.TrimSpace(article.Title))
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
