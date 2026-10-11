package bluesky

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
	"unicode"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

const (
	maxResponseSize = 2 << 20
	maxPosts        = 10
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	cacheTTL   time.Duration
	symbols    []domain.MarketSymbol

	mu    sync.Mutex
	cache map[string]cachedFeed
}

type cachedFeed struct {
	value    domain.SocialFeed
	cachedAt time.Time
}

func NewClient(baseURL string, symbols []domain.MarketSymbol, httpClient *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid Bluesky API URL")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(parsedURL.String(), "/"),
		httpClient: httpClient,
		cacheTTL:   2 * time.Minute,
		symbols:    append([]domain.MarketSymbol(nil), symbols...),
		cache:      make(map[string]cachedFeed),
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

	endpoint, err := url.Parse(c.baseURL + "/xrpc/app.bsky.feed.searchPosts")
	if err != nil {
		return domain.SocialFeed{}, fmt.Errorf("create Bluesky search URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("q", c.searchQuery(symbol))
	query.Set("limit", "25")
	query.Set("sort", "latest")
	query.Set("lang", "en")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return domain.SocialFeed{}, fmt.Errorf("create Bluesky request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "TradingWens/1.0")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return domain.SocialFeed{}, fmt.Errorf("fetch Bluesky posts: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return domain.SocialFeed{}, fmt.Errorf("read Bluesky response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return domain.SocialFeed{}, fmt.Errorf("Bluesky request failed with status %d", res.StatusCode)
	}

	var payload providerResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.SocialFeed{}, fmt.Errorf("decode Bluesky response: %w", err)
	}

	posts := make([]domain.SocialPost, 0, maxPosts)
	for _, item := range payload.Posts {
		post, ok := c.normalizePost(item, symbol)
		if !ok {
			continue
		}
		posts = append(posts, post)
		if len(posts) == maxPosts {
			break
		}
	}
	result := domain.SocialFeed{
		Data:   posts,
		AsOf:   time.Now().UTC(),
		Source: "bluesky",
		Symbol: symbol,
		Count:  len(posts),
	}
	c.cache[symbol] = cachedFeed{value: cloneFeed(result), cachedAt: result.AsOf}
	return result, nil
}

type providerResponse struct {
	Posts []providerPost `json:"posts"`
}

type providerPost struct {
	URI    string `json:"uri"`
	Author struct {
		DID         string `json:"did"`
		Handle      string `json:"handle"`
		DisplayName string `json:"displayName"`
		Avatar      string `json:"avatar"`
	} `json:"author"`
	Record struct {
		Text      string    `json:"text"`
		CreatedAt time.Time `json:"createdAt"`
		Langs     []string  `json:"langs"`
	} `json:"record"`
	IndexedAt   time.Time `json:"indexedAt"`
	LikeCount   int       `json:"likeCount"`
	ReplyCount  int       `json:"replyCount"`
	RepostCount int       `json:"repostCount"`
	QuoteCount  int       `json:"quoteCount"`
}

func (c *Client) normalizePost(item providerPost, symbol string) (domain.SocialPost, bool) {
	text := strings.TrimSpace(item.Record.Text)
	if item.URI == "" || text == "" || !c.stockRelated(text, symbol) {
		return domain.SocialPost{}, false
	}
	createdAt := item.Record.CreatedAt
	if createdAt.IsZero() {
		createdAt = item.IndexedAt
	}
	if createdAt.IsZero() {
		return domain.SocialPost{}, false
	}
	handle := strings.TrimSpace(item.Author.Handle)
	actor := handle
	if actor == "" {
		actor = strings.TrimSpace(item.Author.DID)
	}
	recordKey := recordKeyFromURI(item.URI)
	if actor == "" || recordKey == "" {
		return domain.SocialPost{}, false
	}
	language := ""
	if len(item.Record.Langs) > 0 {
		language = strings.TrimSpace(item.Record.Langs[0])
	}
	return domain.SocialPost{
		ID:            item.URI,
		Text:          text,
		URL:           "https://bsky.app/profile/" + url.PathEscape(actor) + "/post/" + url.PathEscape(recordKey),
		AuthorName:    strings.TrimSpace(item.Author.DisplayName),
		Username:      handle,
		AvatarURL:     strings.TrimSpace(item.Author.Avatar),
		CreatedAt:     createdAt.UTC(),
		Language:      language,
		MatchedSymbol: symbol,
		Metrics: domain.SocialMetrics{
			Likes:   item.LikeCount,
			Replies: item.ReplyCount,
			Reposts: item.RepostCount,
			Quotes:  item.QuoteCount,
		},
	}, true
}

func (c *Client) searchQuery(symbol string) string {
	terms := []string{"$" + symbol, symbol + " stock"}
	for _, candidate := range c.symbols {
		if candidate.Symbol != symbol {
			continue
		}
		terms = append(terms, candidate.Name+" stock")
		for _, alias := range candidate.Aliases {
			terms = append(terms, alias+" stock")
		}
		break
	}
	quoted := make([]string, 0, len(terms))
	for _, term := range terms {
		quoted = append(quoted, quoteTerm(term))
	}
	return strings.Join(quoted, " OR ")
}

func (c *Client) stockRelated(text, symbol string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, strings.ToLower("$"+symbol)) {
		return true
	}
	if !containsMarketTerm(lower) {
		return false
	}
	if containsToken(lower, strings.ToLower(symbol)) {
		return true
	}
	for _, candidate := range c.symbols {
		if candidate.Symbol != symbol {
			continue
		}
		if strings.Contains(lower, strings.ToLower(candidate.Name)) {
			return true
		}
		for _, alias := range candidate.Aliases {
			if strings.Contains(lower, strings.ToLower(alias)) {
				return true
			}
		}
	}
	return false
}

func containsMarketTerm(text string) bool {
	terms := []string{"stock", "stocks", "share", "shares", "earnings", "investor", "market", "bullish", "bearish", "price target", "dividend"}
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func containsToken(text, token string) bool {
	for _, field := range strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '-'
	}) {
		if field == token {
			return true
		}
	}
	return false
}

func recordKeyFromURI(uri string) string {
	parts := strings.Split(strings.TrimSpace(uri), "/")
	if len(parts) < 5 || parts[0] != "at:" || parts[len(parts)-2] != "app.bsky.feed.post" {
		return ""
	}
	return parts[len(parts)-1]
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
