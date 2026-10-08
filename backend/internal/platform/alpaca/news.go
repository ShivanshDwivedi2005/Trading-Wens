package alpaca

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type NewsClient struct {
	baseURL    string
	keyID      string
	secretKey  string
	symbols    []domain.MarketSymbol
	httpClient *http.Client
	cacheTTL   time.Duration

	mu    sync.Mutex
	cache map[string]cachedNews
}

type cachedNews struct {
	feed     domain.NewsFeed
	cachedAt time.Time
}

func NewNewsClient(baseURL, keyID, secretKey string, symbols []domain.MarketSymbol, httpClient *http.Client) (*NewsClient, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("invalid Alpaca data URL")
	}
	if strings.TrimSpace(keyID) == "" || strings.TrimSpace(secretKey) == "" {
		return nil, errors.New("Alpaca API credentials are required")
	}
	if len(symbols) == 0 {
		return nil, errors.New("at least one news symbol is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &NewsClient{
		baseURL:    parsed.String(),
		keyID:      strings.TrimSpace(keyID),
		secretKey:  strings.TrimSpace(secretKey),
		symbols:    append([]domain.MarketSymbol(nil), symbols...),
		httpClient: httpClient,
		cacheTTL:   2 * time.Minute,
		cache:      make(map[string]cachedNews),
	}, nil
}

func (c *NewsClient) Latest(ctx context.Context) (domain.NewsFeed, error) {
	return c.fetch(ctx, "market", symbolNames(c.symbols), 50)
}

func (c *NewsClient) LatestForSymbol(ctx context.Context, requestedSymbol string) (domain.NewsFeed, error) {
	symbol := strings.ToUpper(strings.TrimSpace(requestedSymbol))
	if !validMarketSymbol(symbol) {
		return domain.NewsFeed{}, domain.ErrUnsupportedSymbol
	}
	return c.fetch(ctx, "symbol:"+symbol, []string{symbol}, 25)
}

func (c *NewsClient) fetch(ctx context.Context, cacheKey string, symbols []string, limit int) (domain.NewsFeed, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.cache[cacheKey]; ok && time.Since(cached.cachedAt) < c.cacheTTL {
		return cloneNewsFeed(cached.feed), nil
	}

	endpoint, err := url.Parse(c.baseURL + "/v1beta1/news")
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("create Alpaca news URL: %w", err)
	}
	now := time.Now().UTC()
	query := endpoint.Query()
	query.Set("symbols", strings.Join(symbols, ","))
	query.Set("start", now.Add(-72*time.Hour).Format(time.RFC3339))
	query.Set("end", now.Format(time.RFC3339))
	query.Set("limit", strconv.Itoa(limit))
	query.Set("sort", "desc")
	query.Set("include_content", "false")
	query.Set("exclude_contentless", "true")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("create Alpaca news request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("APCA-API-KEY-ID", c.keyID)
	req.Header.Set("APCA-API-SECRET-KEY", c.secretKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("fetch Alpaca news: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("read Alpaca news response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return domain.NewsFeed{}, fmt.Errorf("Alpaca news request failed with status %d", res.StatusCode)
	}

	var payload alpacaNewsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.NewsFeed{}, fmt.Errorf("decode Alpaca news response: %w", err)
	}
	articles := make([]domain.NewsArticle, 0, len(payload.News))
	for _, providerArticle := range payload.News {
		if article, ok := normalizeNewsArticle(providerArticle); ok {
			articles = append(articles, article)
		}
	}
	sort.SliceStable(articles, func(i, j int) bool {
		return articles[i].PublishedAt.After(articles[j].PublishedAt)
	})
	feed := domain.NewsFeed{
		Data:      articles,
		AsOf:      now,
		Source:    "alpaca",
		Providers: []string{"alpaca"},
		Count:     len(articles),
	}
	c.cache[cacheKey] = cachedNews{feed: cloneNewsFeed(feed), cachedAt: now}
	return feed, nil
}

type alpacaNewsResponse struct {
	News []alpacaNewsArticle `json:"news"`
}

type alpacaNewsArticle struct {
	ID        int64             `json:"id"`
	Headline  string            `json:"headline"`
	Summary   string            `json:"summary"`
	URL       string            `json:"url"`
	Source    string            `json:"source"`
	CreatedAt time.Time         `json:"created_at"`
	Images    []alpacaNewsImage `json:"images"`
	Symbols   []string          `json:"symbols"`
}

type alpacaNewsImage struct {
	Size string `json:"size"`
	URL  string `json:"url"`
}

func normalizeNewsArticle(provider alpacaNewsArticle) (domain.NewsArticle, bool) {
	articleURL, err := url.Parse(strings.TrimSpace(provider.URL))
	if err != nil || articleURL.Host == "" || (articleURL.Scheme != "http" && articleURL.Scheme != "https") {
		return domain.NewsArticle{}, false
	}
	title := strings.TrimSpace(provider.Headline)
	if title == "" || provider.CreatedAt.IsZero() {
		return domain.NewsArticle{}, false
	}
	symbols := make([]string, 0, len(provider.Symbols))
	for _, symbol := range provider.Symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if validMarketSymbol(symbol) {
			symbols = append(symbols, symbol)
		}
	}
	sort.Strings(symbols)
	return domain.NewsArticle{
		ID:             "alpaca-" + strconv.FormatInt(provider.ID, 10),
		Title:          title,
		Summary:        strings.TrimSpace(provider.Summary),
		URL:            articleURL.String(),
		Domain:         articleURL.Hostname(),
		Provider:       "alpaca",
		PublishedAt:    provider.CreatedAt.UTC(),
		Language:       "English",
		ImageURL:       preferredNewsImage(provider.Images),
		MatchedSymbols: symbols,
	}, true
}

func preferredNewsImage(images []alpacaNewsImage) string {
	for _, preferredSize := range []string{"large", "medium", "small"} {
		for _, image := range images {
			if image.Size == preferredSize && validHTTPImageURL(image.URL) {
				return image.URL
			}
		}
	}
	return ""
}

func validHTTPImageURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func cloneNewsFeed(source domain.NewsFeed) domain.NewsFeed {
	cloned := source
	cloned.Providers = append([]string(nil), source.Providers...)
	cloned.Data = append([]domain.NewsArticle(nil), source.Data...)
	for index := range cloned.Data {
		cloned.Data[index].MatchedSymbols = append([]string(nil), source.Data[index].MatchedSymbols...)
	}
	return cloned
}
