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

	mu           sync.Mutex
	cached       domain.MarketSnapshotSet
	cachedAt     time.Time
	historyCache map[string]cachedHistory
}

type cachedHistory struct {
	value    domain.StockHistory
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
		baseURL:      parsedURL.String(),
		keyID:        strings.TrimSpace(keyID),
		secretKey:    strings.TrimSpace(secretKey),
		feed:         valueOrDefault(strings.TrimSpace(feed), "iex"),
		symbols:      append([]domain.MarketSymbol(nil), symbols...),
		httpClient:   httpClient,
		cacheTTL:     10 * time.Second,
		historyCache: make(map[string]cachedHistory),
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

func (c *Client) History(ctx context.Context, requestedSymbol, requestedRange string) (domain.StockHistory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	symbol, ok := c.findSymbol(requestedSymbol)
	if !ok {
		return domain.StockHistory{}, domain.ErrUnsupportedSymbol
	}
	historyRange, timeframe, lookback, limit, err := historyWindow(requestedRange)
	if err != nil {
		return domain.StockHistory{}, err
	}
	cacheKey := symbol.Symbol + ":" + historyRange
	if cached, ok := c.historyCache[cacheKey]; ok && time.Since(cached.cachedAt) < 30*time.Second {
		return cloneHistory(cached.value), nil
	}

	endpoint, err := url.Parse(c.baseURL + "/v2/stocks/" + url.PathEscape(symbol.Symbol) + "/bars")
	if err != nil {
		return domain.StockHistory{}, fmt.Errorf("create Alpaca bars URL: %w", err)
	}
	now := time.Now().UTC()
	query := endpoint.Query()
	query.Set("feed", c.feed)
	query.Set("timeframe", timeframe)
	query.Set("start", now.Add(-lookback).Format(time.RFC3339))
	query.Set("end", now.Format(time.RFC3339))
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("sort", "asc")
	query.Set("adjustment", "raw")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return domain.StockHistory{}, fmt.Errorf("create Alpaca bars request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("APCA-API-KEY-ID", c.keyID)
	req.Header.Set("APCA-API-SECRET-KEY", c.secretKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return domain.StockHistory{}, fmt.Errorf("fetch Alpaca bars: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return domain.StockHistory{}, fmt.Errorf("read Alpaca bars response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return domain.StockHistory{}, fmt.Errorf("Alpaca bars request failed with status %d", res.StatusCode)
	}

	var payload providerBarsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return domain.StockHistory{}, fmt.Errorf("decode Alpaca bars: %w", err)
	}
	bars := make([]domain.MarketBar, 0, len(payload.Bars))
	for _, bar := range payload.Bars {
		if bar.Timestamp.IsZero() || bar.Close <= 0 {
			continue
		}
		bars = append(bars, domain.MarketBar{
			Timestamp: bar.Timestamp,
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			Volume:    bar.Volume,
		})
	}
	if historyRange == "1D" && len(bars) > 0 {
		latestSession := bars[len(bars)-1].Timestamp.Format("2006-01-02")
		firstBar := 0
		for firstBar < len(bars) && bars[firstBar].Timestamp.Format("2006-01-02") != latestSession {
			firstBar++
		}
		bars = bars[firstBar:]
	}
	asOf := now
	if len(bars) > 0 {
		asOf = bars[len(bars)-1].Timestamp
	}
	result := domain.StockHistory{
		Symbol:    symbol.Symbol,
		Name:      symbol.Name,
		Data:      bars,
		AsOf:      asOf,
		Range:     historyRange,
		Timeframe: timeframe,
		Feed:      c.feed,
		Source:    "alpaca",
		Count:     len(bars),
	}
	c.historyCache[cacheKey] = cachedHistory{value: cloneHistory(result), cachedAt: now}
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

type providerBarsResponse struct {
	Bars []providerBar `json:"bars"`
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

func (c *Client) findSymbol(value string) (domain.MarketSymbol, bool) {
	requested := strings.ToUpper(strings.TrimSpace(value))
	for _, symbol := range c.symbols {
		if symbol.Symbol == requested {
			return symbol, true
		}
	}
	return domain.MarketSymbol{}, false
}

func historyWindow(value string) (string, string, time.Duration, int, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "1D":
		return "1D", "5Min", 72 * time.Hour, 500, nil
	case "5D":
		return "5D", "15Min", 10 * 24 * time.Hour, 1000, nil
	case "1M":
		return "1M", "1Hour", 45 * 24 * time.Hour, 1000, nil
	default:
		return "", "", 0, 0, domain.ErrUnsupportedRange
	}
}

func cloneHistory(source domain.StockHistory) domain.StockHistory {
	cloned := source
	cloned.Data = append([]domain.MarketBar(nil), source.Data...)
	return cloned
}
