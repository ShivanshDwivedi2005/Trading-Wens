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

	mu       sync.Mutex
	cached   domain.NewsFeed
	cachedAt time.Time
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid GDELT API URL")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		baseURL:    parsedURL.String(),
		httpClient: httpClient,
		cacheTTL:   2 * time.Minute,
	}, nil
}

func (c *Client) Latest(ctx context.Context) (domain.NewsFeed, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.cachedAt.IsZero() && time.Since(c.cachedAt) < c.cacheTTL {
		return cloneFeed(c.cached), nil
	}

	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("create GDELT URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("query", marketQuery())
	query.Set("mode", "artlist")
	query.Set("format", "json")
	query.Set("maxrecords", "50")
	query.Set("timespan", "24h")
	query.Set("sort", "datedesc")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return domain.NewsFeed{}, fmt.Errorf("create GDELT request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "TradingWens/1.0")

	res, err := c.httpClient.Do(req)
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
		article, ok := normalizeArticle(providerArticle)
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
		Data:   articles,
		AsOf:   time.Now().UTC(),
		Source: "gdelt",
		Count:  len(articles),
	}
	c.cached = cloneFeed(result)
	c.cachedAt = result.AsOf
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

func normalizeArticle(provider providerArticle) (domain.NewsArticle, bool) {
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
	return domain.NewsArticle{
		ID:             hex.EncodeToString(digest[:8]),
		Title:          title,
		URL:            articleURL.String(),
		Domain:         strings.TrimSpace(provider.Domain),
		PublishedAt:    publishedAt,
		Language:       strings.TrimSpace(provider.Language),
		SourceCountry:  strings.TrimSpace(provider.SourceCountry),
		ImageURL:       imageURL,
		MatchedSymbols: matchSymbols(title),
	}, true
}

func marketQuery() string {
	return `("NVIDIA" OR "Apple" OR "Alphabet" OR "Microsoft" OR "Amazon" OR "Broadcom" OR "Meta Platforms" OR "Tesla" OR "Berkshire Hathaway" OR "Eli Lilly" OR "JPMorgan" OR "Walmart" OR "Visa" OR "Oracle" OR "Exxon Mobil" OR "Johnson & Johnson" OR "Mastercard" OR "Netflix" OR "Costco" OR "AbbVie" OR "Home Depot" OR "Procter & Gamble" OR "Bank of America" OR "GE Aerospace" OR "Coca-Cola" OR "Cisco" OR "Caterpillar" OR "Philip Morris" OR "IBM" OR "Chevron") sourcelang:english`
}

func matchSymbols(title string) []string {
	lowerTitle := strings.ToLower(title)
	aliases := map[string][]string{
		"AAPL": {"apple"}, "ABBV": {"abbvie"}, "AMZN": {"amazon"}, "AVGO": {"broadcom"},
		"BAC": {"bank of america"}, "BRK.B": {"berkshire hathaway"}, "CAT": {"caterpillar"},
		"COST": {"costco"}, "CSCO": {"cisco"}, "CVX": {"chevron"}, "GE": {"ge aerospace"},
		"GOOGL": {"alphabet", "google"}, "HD": {"home depot"}, "IBM": {"ibm"},
		"JNJ": {"johnson & johnson", "johnson and johnson"}, "JPM": {"jpmorgan"},
		"KO": {"coca-cola", "coca cola"}, "LLY": {"eli lilly"}, "MA": {"mastercard"},
		"META": {"meta platforms", "facebook"}, "MSFT": {"microsoft"}, "NFLX": {"netflix"},
		"NVDA": {"nvidia"}, "ORCL": {"oracle"}, "PG": {"procter & gamble", "procter and gamble"},
		"PM": {"philip morris"}, "TSLA": {"tesla"}, "V": {"visa"}, "WMT": {"walmart"},
		"XOM": {"exxon mobil", "exxonmobil"},
	}

	matches := make([]string, 0, 2)
	for symbol, names := range aliases {
		for _, name := range names {
			if strings.Contains(lowerTitle, name) {
				matches = append(matches, symbol)
				break
			}
		}
	}
	sort.Strings(matches)
	return matches
}

func cloneFeed(source domain.NewsFeed) domain.NewsFeed {
	cloned := source
	cloned.Data = append([]domain.NewsArticle(nil), source.Data...)
	for index := range cloned.Data {
		cloned.Data[index].MatchedSymbols = append([]string(nil), source.Data[index].MatchedSymbols...)
	}
	return cloned
}
