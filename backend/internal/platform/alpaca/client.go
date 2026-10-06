package alpaca

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

const maxResponseSize = 4 << 20

type Client struct {
	baseURL    string
	keyID      string
	secretKey  string
	feed       string
	symbols    []domain.MarketSymbol
	httpClient *http.Client
	cacheTTL   time.Duration

	mu       sync.Mutex
	cached   domain.MarketSnapshotSet
	cachedAt time.Time
}

func NewClient(baseURL, keyID, secretKey, feed string, symbols []domain.MarketSymbol, httpClient *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid Alpaca data URL")
	}
	if strings.TrimSpace(keyID) == "" || strings.TrimSpace(secretKey) == "" {
		return nil, errors.New("Alpaca API credentials are required")
	}
	if len(symbols) == 0 {
		return nil, errors.New("at least one market symbol is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		baseURL:    parsedURL.String(),
		keyID:      strings.TrimSpace(keyID),
		secretKey:  strings.TrimSpace(secretKey),
		feed:       valueOrDefault(strings.TrimSpace(feed), "iex"),
		symbols:    append([]domain.MarketSymbol(nil), symbols...),
		httpClient: httpClient,
		cacheTTL:   10 * time.Second,
	}, nil
}

func (c *Client) Snapshots(ctx context.Context) (domain.MarketSnapshotSet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.cachedAt.IsZero() && time.Since(c.cachedAt) < c.cacheTTL {
		return cloneSnapshotSet(c.cached), nil
	}

	endpoint, err := url.Parse(c.baseURL + "/v2/stocks/snapshots")
	if err != nil {
		return domain.MarketSnapshotSet{}, fmt.Errorf("create Alpaca snapshots URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("symbols", strings.Join(symbolNames(c.symbols), ","))
	query.Set("feed", c.feed)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return domain.MarketSnapshotSet{}, fmt.Errorf("create Alpaca request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("APCA-API-KEY-ID", c.keyID)
	req.Header.Set("APCA-API-SECRET-KEY", c.secretKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return domain.MarketSnapshotSet{}, fmt.Errorf("fetch Alpaca snapshots: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return domain.MarketSnapshotSet{}, fmt.Errorf("read Alpaca response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return domain.MarketSnapshotSet{}, fmt.Errorf("Alpaca snapshots request failed with status %d", res.StatusCode)
	}

	var providerSnapshots map[string]providerSnapshot
	if err := json.Unmarshal(body, &providerSnapshots); err != nil {
		return domain.MarketSnapshotSet{}, fmt.Errorf("decode Alpaca snapshots: %w", err)
	}

	now := time.Now().UTC()
	result := domain.MarketSnapshotSet{
		Data:   make([]domain.MarketSnapshot, 0, len(c.symbols)),
		AsOf:   now,
		Feed:   c.feed,
		Source: "alpaca",
		Count:  len(c.symbols),
	}
	var latest time.Time
	for _, symbol := range c.symbols {
		snapshot := normalizeSnapshot(symbol, providerSnapshots[symbol.Symbol])
		if snapshot.Timestamp.After(latest) {
			latest = snapshot.Timestamp
		}
		result.Data = append(result.Data, snapshot)
	}
	if !latest.IsZero() {
		result.AsOf = latest
	}

	c.cached = cloneSnapshotSet(result)
	c.cachedAt = now
	return result, nil
}

type providerSnapshot struct {
	LatestTrade      providerTrade `json:"latestTrade"`
	MinuteBar        providerBar   `json:"minuteBar"`
	DailyBar         providerBar   `json:"dailyBar"`
	PreviousDailyBar providerBar   `json:"prevDailyBar"`
}

type providerTrade struct {
	Timestamp time.Time `json:"t"`
	Price     float64   `json:"p"`
}

type providerBar struct {
	Timestamp time.Time `json:"t"`
	Open      float64   `json:"o"`
	High      float64   `json:"h"`
	Low       float64   `json:"l"`
	Close     float64   `json:"c"`
	Volume    uint64    `json:"v"`
}

func normalizeSnapshot(symbol domain.MarketSymbol, provider providerSnapshot) domain.MarketSnapshot {
	price := provider.LatestTrade.Price
	timestamp := provider.LatestTrade.Timestamp
	if price <= 0 {
		price = provider.MinuteBar.Close
		timestamp = provider.MinuteBar.Timestamp
	}
	if price <= 0 {
		price = provider.DailyBar.Close
		timestamp = provider.DailyBar.Timestamp
	}

	previousClose := provider.PreviousDailyBar.Close
	change := 0.0
	changePercent := 0.0
	if price > 0 && previousClose > 0 {
		change = price - previousClose
		changePercent = change / previousClose * 100
	}

	return domain.MarketSnapshot{
		Symbol:        symbol.Symbol,
		Name:          symbol.Name,
		Price:         price,
		Change:        change,
		ChangePercent: changePercent,
		Open:          provider.DailyBar.Open,
		High:          provider.DailyBar.High,
		Low:           provider.DailyBar.Low,
		PreviousClose: previousClose,
		Volume:        provider.DailyBar.Volume,
		Timestamp:     timestamp,
		Available:     price > 0,
	}
}

func symbolNames(symbols []domain.MarketSymbol) []string {
	names := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		names = append(names, symbol.Symbol)
	}
	return names
}

func cloneSnapshotSet(source domain.MarketSnapshotSet) domain.MarketSnapshotSet {
	cloned := source
	cloned.Data = append([]domain.MarketSnapshot(nil), source.Data...)
	return cloned
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
