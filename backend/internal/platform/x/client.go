package x

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

const maxResponseSize = 2 << 20

type Client struct {
	baseURL     string
	bearerToken string
	httpClient  *http.Client
	cacheTTL    time.Duration
	symbols     []domain.MarketSymbol

	mu    sync.Mutex
	cache map[string]cachedFeed
}

type cachedFeed struct {
	value    domain.SocialFeed
	cachedAt time.Time
}

func NewClient(baseURL, bearerToken string, symbols []domain.MarketSymbol, httpClient *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid X API URL")
	}
	if strings.TrimSpace(bearerToken) == "" {
		return nil, errors.New("X bearer token is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:     strings.TrimRight(parsedURL.String(), "/"),
		bearerToken: strings.TrimSpace(bearerToken),
		httpClient:  httpClient,
		cacheTTL:    2 * time.Minute,
		symbols:     append([]domain.MarketSymbol(nil), symbols...),
		cache:       make(map[string]cachedFeed),
	}, nil
}

func (c *Client) LatestForSymbol(ctx context.Context, requestedSymbol string) (domain.SocialFeed, error) {
	symbol := strings.ToUpper(strings.TrimSpace(requestedSymbol))
	if !validTicker(symbol) {
		return domain.SocialFeed{}, domain.ErrUnsupportedSymbol
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.cache[symbol]; ok && time.Since(cached.cachedAt) < c.cacheTTL {
		return cloneFeed(cached.value), nil
	}

	endpoint, err := url.Parse(c.baseURL + "/2/tweets/search/recent")
	if err != nil {
		return domain.SocialFeed{}, fmt.Errorf("create X search URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("query", c.searchQuery(symbol))
	query.Set("max_results", "10")
	query.Set("sort_order", "recency")
	query.Set("post.fields", "author_id,created_at,lang,public_metrics")
	query.Set("expansions", "author_id")
	query.Set("user.fields", "name,username,profile_image_url")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return domain.SocialFeed{}, fmt.Errorf("create X request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	req.Header.Set("User-Agent", "TradingWens/1.0")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return domain.SocialFeed{}, fmt.Errorf("fetch X posts: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return domain.SocialFeed{}, fmt.Errorf("read X response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return domain.SocialFeed{}, fmt.Errorf("X request failed with status %d", res.StatusCode)
	}

	var payload providerResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.SocialFeed{}, fmt.Errorf("decode X response: %w", err)
	}
	users := make(map[string]providerUser, len(payload.Includes.Users))
	for _, user := range payload.Includes.Users {
		users[user.ID] = user
	}

	posts := make([]domain.SocialPost, 0, len(payload.Data))
	for _, item := range payload.Data {
		post, ok := normalizePost(item, users[item.AuthorID], symbol)
		if ok {
			posts = append(posts, post)
		}
	}
	result := domain.SocialFeed{
		Data:   posts,
		AsOf:   time.Now().UTC(),
		Source: "x",
		Symbol: symbol,
		Count:  len(posts),
	}
	c.cache[symbol] = cachedFeed{value: cloneFeed(result), cachedAt: result.AsOf}
	return result, nil
}

type providerResponse struct {
	Data     []providerPost `json:"data"`
	Includes struct {
		Users []providerUser `json:"users"`
	} `json:"includes"`
}

type providerPost struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	Lang      string    `json:"lang"`
	Metrics   struct {
		LikeCount    int `json:"like_count"`
		ReplyCount   int `json:"reply_count"`
		RepostCount  int `json:"repost_count"`
		RetweetCount int `json:"retweet_count"`
		QuoteCount   int `json:"quote_count"`
	} `json:"public_metrics"`
}

type providerUser struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url"`
}

func normalizePost(item providerPost, author providerUser, symbol string) (domain.SocialPost, bool) {
	text := strings.TrimSpace(item.Text)
	if item.ID == "" || text == "" || item.CreatedAt.IsZero() {
		return domain.SocialPost{}, false
	}
	username := strings.TrimSpace(author.Username)
	postURL := "https://x.com/i/web/status/" + item.ID
	if username != "" {
		postURL = "https://x.com/" + url.PathEscape(username) + "/status/" + item.ID
	}
	reposts := item.Metrics.RepostCount
	if reposts == 0 {
		reposts = item.Metrics.RetweetCount
	}
	return domain.SocialPost{
		ID:            item.ID,
		Text:          text,
		URL:           postURL,
		AuthorName:    strings.TrimSpace(author.Name),
		Username:      username,
		AvatarURL:     strings.TrimSpace(author.ProfileImageURL),
		CreatedAt:     item.CreatedAt.UTC(),
		Language:      strings.TrimSpace(item.Lang),
		MatchedSymbol: symbol,
		Metrics: domain.SocialMetrics{
			Likes:   item.Metrics.LikeCount,
			Replies: item.Metrics.ReplyCount,
			Reposts: reposts,
			Quotes:  item.Metrics.QuoteCount,
		},
	}, true
}

func (c *Client) searchQuery(symbol string) string {
	terms := []string{"$" + symbol}
	for _, candidate := range c.symbols {
		if candidate.Symbol != symbol {
			continue
		}
		terms = append(terms, quoteTerm(candidate.Name))
		for _, alias := range candidate.Aliases {
			terms = append(terms, quoteTerm(alias))
		}
		break
	}
	return "(" + strings.Join(terms, " OR ") + ") (stock OR shares OR earnings OR market) lang:en -is:retweet"
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

func quoteTerm(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, "") + `"`
}

func cloneFeed(source domain.SocialFeed) domain.SocialFeed {
	cloned := source
	cloned.Data = append([]domain.SocialPost(nil), source.Data...)
	return cloned
}
