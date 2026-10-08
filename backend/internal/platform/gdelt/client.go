package gdelt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

const maxResponseSize = 4 << 20

type Client struct {
	baseURL    string
	httpClient *http.Client
	cacheTTL   time.Duration
	symbols    []domain.MarketSymbol

	mu            sync.Mutex
	cache         map[string]cachedFeed
	lastRequestAt time.Time
}

type cachedFeed struct {
	value    domain.NewsFeed
	cachedAt time.Time
}

func NewClient(baseURL string, symbols []domain.MarketSymbol, httpClient *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid GDELT API URL")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	if len(symbols) == 0 {
		return nil, errors.New("at least one news symbol is required")
	}
	return &Client{
		baseURL:    parsedURL.String(),
		httpClient: httpClient,
		cacheTTL:   2 * time.Minute,
		symbols:    append([]domain.MarketSymbol(nil), symbols...),
		cache:      make(map[string]cachedFeed),
	}, nil
}

func (c *Client) Latest(ctx context.Context) (domain.NewsFeed, error) {
	return c.fetch(ctx, "market", marketQuery(c.symbols), "24h", "50", "")
}

func (c *Client) LatestForSymbol(ctx context.Context, requestedSymbol string) (domain.NewsFeed, error) {
	requestedSymbol = strings.ToUpper(strings.TrimSpace(requestedSymbol))
	symbol, ok := c.findSymbol(requestedSymbol)
	if !ok {
		if !validTicker(requestedSymbol) {
			return domain.NewsFeed{}, domain.ErrUnsupportedSymbol
		}
		return c.fetch(ctx, "symbol:"+requestedSymbol, tickerQuery(requestedSymbol), "3d", "25", requestedSymbol)
	}
	return c.fetch(ctx, "symbol:"+symbol.Symbol, companyQuery(symbol), "3d", "25", symbol.Symbol)
}

func (c *Client) fetch(ctx context.Context, cacheKey, queryText, timespan, maxRecords, forcedSymbol string) (domain.NewsFeed, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.cache[cacheKey]; ok && time.Since(cached.cachedAt) < c.cacheTTL {
		return cloneFeed(cached.value), nil
	}
	if wait := 5*time.Second - time.Since(c.lastRequestAt); !c.lastRequestAt.IsZero() && wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return domain.NewsFeed{}, ctx.Err()
		case <-timer.C:
		}
	}

	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("create GDELT URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("query", queryText)
	query.Set("mode", "artlist")
	query.Set("format", "json")
	query.Set("maxrecords", maxRecords)
	query.Set("timespan", timespan)
	query.Set("sort", "datedesc")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("create GDELT request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "TradingWens/1.0")

	res, err := c.httpClient.Do(req)
	c.lastRequestAt = time.Now()
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("fetch GDELT news: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("read GDELT response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return domain.NewsFeed{}, fmt.Errorf("GDELT request failed with status %d", res.StatusCode)
	}

	var payload providerResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.NewsFeed{}, fmt.Errorf("decode GDELT response: %w", err)
	}

	articles := make([]domain.NewsArticle, 0, len(payload.Articles))
	seen := make(map[string]struct{}, len(payload.Articles))
	for _, providerArticle := range payload.Articles {
		article, ok := normalizeArticle(providerArticle, c.symbols, forcedSymbol)
		if !ok {
			continue
		}
		if _, exists := seen[article.URL]; exists {
			continue
		}
		seen[article.URL] = struct{}{}
		articles = append(articles, article)
	}

	result := domain.NewsFeed{
		Data:      articles,
		AsOf:      time.Now().UTC(),
		Source:    "gdelt",
		Providers: []string{"gdelt"},
		Count:     len(articles),
	}
	c.cache[cacheKey] = cachedFeed{value: cloneFeed(result), cachedAt: result.AsOf}
	return result, nil
}

type providerResponse struct {
	Articles []providerArticle `json:"articles"`
}

type providerArticle struct {
	URL           string `json:"url"`
	Title         string `json:"title"`
	SeenDate      string `json:"seendate"`
	SocialImage   string `json:"socialimage"`
	Domain        string `json:"domain"`
	Language      string `json:"language"`
	SourceCountry string `json:"sourcecountry"`
}

func normalizeArticle(provider providerArticle, symbols []domain.MarketSymbol, forcedSymbol string) (domain.NewsArticle, bool) {
	articleURL, err := url.Parse(strings.TrimSpace(provider.URL))
	if err != nil || (articleURL.Scheme != "http" && articleURL.Scheme != "https") || articleURL.Host == "" {
		return domain.NewsArticle{}, false
	}
	publishedAt, err := time.Parse("20060102T150405Z", provider.SeenDate)
	if err != nil {
		return domain.NewsArticle{}, false
	}
	title := strings.TrimSpace(provider.Title)
	if title == "" {
		return domain.NewsArticle{}, false
	}

	imageURL := ""
	if parsedImage, err := url.Parse(strings.TrimSpace(provider.SocialImage)); err == nil && parsedImage.Host != "" && (parsedImage.Scheme == "http" || parsedImage.Scheme == "https") {
		imageURL = parsedImage.String()
	}
	digest := sha256.Sum256([]byte(articleURL.String()))
	matchedSymbols := matchSymbols(title, symbols)
	if forcedSymbol != "" && !containsSymbol(matchedSymbols, forcedSymbol) {
		matchedSymbols = append(matchedSymbols, forcedSymbol)
		sort.Strings(matchedSymbols)
	}
	return domain.NewsArticle{
		ID:             hex.EncodeToString(digest[:8]),
		Title:          title,
		URL:            articleURL.String(),
		Domain:         strings.TrimSpace(provider.Domain),
		Provider:       "gdelt",
		PublishedAt:    publishedAt,
		Language:       strings.TrimSpace(provider.Language),
		SourceCountry:  strings.TrimSpace(provider.SourceCountry),
		ImageURL:       imageURL,
		MatchedSymbols: matchedSymbols,
	}, true
}

func marketQuery(symbols []domain.MarketSymbol) string {
	terms := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		terms = append(terms, quoteTerm(symbol.Name))
	}
	return "(" + strings.Join(terms, " OR ") + ") sourcelang:english"
}

func companyQuery(symbol domain.MarketSymbol) string {
	terms := []string{quoteTerm(symbol.Name), quoteTerm(symbol.Symbol)}
	for _, alias := range symbol.Aliases {
		terms = append(terms, quoteTerm(alias))
	}
	return "(" + strings.Join(terms, " OR ") + ") (stock OR shares OR earnings OR company) sourcelang:english"
}

func tickerQuery(symbol string) string {
	return "(" + quoteTerm(symbol) + ") (stock OR shares OR earnings OR company) sourcelang:english"
}

func validTicker(value string) bool {
	if len(value) == 0 || len(value) > 15 {
		return false
	}
	for index, character := range value {
		if character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || (index > 0 && (character == '.' || character == '-')) {
			continue
		}
		return false
	}
	return value[0] >= 'A' && value[0] <= 'Z'
}

func matchSymbols(title string, symbols []domain.MarketSymbol) []string {
	lowerTitle := strings.ToLower(title)
	matches := make([]string, 0, 2)
	for _, symbol := range symbols {
		names := append([]string{symbol.Name}, symbol.Aliases...)
		for _, name := range names {
			if strings.Contains(lowerTitle, strings.ToLower(name)) {
				matches = append(matches, symbol.Symbol)
				break
			}
		}
	}
	sort.Strings(matches)
	return matches
}

func (c *Client) findSymbol(value string) (domain.MarketSymbol, bool) {
	requested := strings.ToUpper(strings.TrimSpace(value))
	for _, symbol := range c.symbols {
		if symbol.Symbol == requested {
			return symbol, true
		}
	}
	return domain.MarketSymbol{}, false
}

func quoteTerm(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, "") + `"`
}

func containsSymbol(symbols []string, expected string) bool {
	for _, symbol := range symbols {
		if symbol == expected {
			return true
		}
	}
	return false
}

func cloneFeed(source domain.NewsFeed) domain.NewsFeed {
	cloned := source
	cloned.Data = append([]domain.NewsArticle(nil), source.Data...)
	for index := range cloned.Data {
		cloned.Data[index].MatchedSymbols = append([]string(nil), source.Data[index].MatchedSymbols...)
	}
	return cloned
}
