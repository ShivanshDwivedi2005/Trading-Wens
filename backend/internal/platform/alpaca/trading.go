package alpaca

import (
	"bytes"
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

type TradingClient struct {
	baseURL    string
	keyID      string
	secretKey  string
	httpClient *http.Client

	mu              sync.Mutex
	portfolio       domain.Portfolio
	portfolioCached time.Time
	assets          []domain.TradingAsset
	assetsCached    time.Time
}

type TradingAPIError struct {
	Status  int
	Message string
}

func (e *TradingAPIError) Error() string {
	return e.Message
}

func NewTradingClient(baseURL, keyID, secretKey string, httpClient *http.Client) (*TradingClient, error) {
	parsedURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("invalid Alpaca trading URL")
	}
	if strings.TrimSpace(keyID) == "" || strings.TrimSpace(secretKey) == "" {
		return nil, errors.New("Alpaca API credentials are required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &TradingClient{
		baseURL:    parsedURL.String(),
		keyID:      strings.TrimSpace(keyID),
		secretKey:  strings.TrimSpace(secretKey),
		httpClient: httpClient,
	}, nil
}

func (c *TradingClient) Portfolio(ctx context.Context) (domain.Portfolio, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.portfolioCached.IsZero() && time.Since(c.portfolioCached) < 5*time.Second {
		return clonePortfolio(c.portfolio), nil
	}

	var account providerAccount
	if err := c.get(ctx, "/v2/account", nil, &account); err != nil {
		return domain.Portfolio{}, err
	}
	var providerPositions []providerPosition
	if err := c.get(ctx, "/v2/positions", nil, &providerPositions); err != nil {
		return domain.Portfolio{}, err
	}

	positions := make([]domain.Position, 0, len(providerPositions))
	for _, position := range providerPositions {
		positions = append(positions, normalizePosition(position))
	}
	sort.Slice(positions, func(i, j int) bool {
		return abs(positions[i].MarketValue) > abs(positions[j].MarketValue)
	})
	result := domain.Portfolio{
		Account:   normalizeAccount(account),
		Positions: positions,
		AsOf:      time.Now().UTC(),
		Source:    "alpaca",
		Mode:      "paper",
	}
	c.portfolio = clonePortfolio(result)
	c.portfolioCached = result.AsOf
	return result, nil
}

func (c *TradingClient) Assets(ctx context.Context, search string) ([]domain.TradingAsset, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.assetsCached.IsZero() || time.Since(c.assetsCached) >= 10*time.Minute {
		query := url.Values{"status": {"active"}, "asset_class": {"us_equity"}}
		var providerAssets []providerAsset
		if err := c.get(ctx, "/v2/assets", query, &providerAssets); err != nil {
			return nil, err
		}
		assets := make([]domain.TradingAsset, 0, len(providerAssets))
		for _, asset := range providerAssets {
			if !asset.Tradable || asset.Status != "active" {
				continue
			}
			assets = append(assets, normalizeAsset(asset))
		}
		sort.Slice(assets, func(i, j int) bool { return assets[i].Symbol < assets[j].Symbol })
		c.assets = assets
		c.assetsCached = time.Now()
	}

	search = strings.ToLower(strings.TrimSpace(search))
	result := make([]domain.TradingAsset, 0, 100)
	for _, asset := range c.assets {
		if search != "" && !strings.Contains(strings.ToLower(asset.Symbol+" "+asset.Name), search) {
			continue
		}
		result = append(result, asset)
		if len(result) == 100 {
			break
		}
	}
	return append([]domain.TradingAsset(nil), result...), nil
}

func (c *TradingClient) SubmitOrder(ctx context.Context, request domain.OrderRequest) (domain.Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var asset providerAsset
	if err := c.get(ctx, "/v2/assets/"+url.PathEscape(request.Symbol), nil, &asset); err != nil {
		return domain.Order{}, err
	}
	if !asset.Tradable || asset.Status != "active" {
		return domain.Order{}, &TradingAPIError{Status: http.StatusUnprocessableEntity, Message: "This asset is not currently tradable"}
	}

	payload := map[string]string{
		"symbol":        request.Symbol,
		"qty":           decimalString(request.Quantity),
		"side":          request.Side,
		"type":          request.Type,
		"time_in_force": request.TimeInForce,
	}
	if request.Type == "limit" {
		payload["limit_price"] = decimalString(request.LimitPrice)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.Order{}, fmt.Errorf("encode Alpaca order: %w", err)
	}
	var providerOrder providerOrder
	if err := c.send(ctx, http.MethodPost, "/v2/orders", body, &providerOrder); err != nil {
		return domain.Order{}, err
	}
	c.portfolioCached = time.Time{}
	return normalizeOrder(providerOrder), nil
}

func (c *TradingClient) get(ctx context.Context, path string, query url.Values, destination any) error {
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("create Alpaca trading URL: %w", err)
	}
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}
	return c.request(ctx, http.MethodGet, endpoint.String(), nil, destination)
}

func (c *TradingClient) send(ctx context.Context, method, path string, body []byte, destination any) error {
	return c.request(ctx, method, c.baseURL+path, body, destination)
}

func (c *TradingClient) request(ctx context.Context, method, endpoint string, body []byte, destination any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return fmt.Errorf("create Alpaca trading request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("APCA-API-KEY-ID", c.keyID)
	req.Header.Set("APCA-API-SECRET-KEY", c.secretKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send Alpaca trading request: %w", err)
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("read Alpaca trading response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		var payload struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(responseBody, &payload)
		message := strings.TrimSpace(payload.Message)
		if message == "" {
			message = "Alpaca rejected the request"
		}
		return &TradingAPIError{Status: res.StatusCode, Message: message}
	}
	if err := json.Unmarshal(responseBody, destination); err != nil {
		return fmt.Errorf("decode Alpaca trading response: %w", err)
	}
	return nil
}

type providerAccount struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	Currency         string `json:"currency"`
	Cash             string `json:"cash"`
	BuyingPower      string `json:"buying_power"`
	PortfolioValue   string `json:"portfolio_value"`
	Equity           string `json:"equity"`
	LastEquity       string `json:"last_equity"`
	LongMarketValue  string `json:"long_market_value"`
	TradingBlocked   bool   `json:"trading_blocked"`
	PatternDayTrader bool   `json:"pattern_day_trader"`
}

type providerPosition struct {
	Symbol            string `json:"symbol"`
	AssetID           string `json:"asset_id"`
	Side              string `json:"side"`
	Qty               string `json:"qty"`
	AverageEntryPrice string `json:"avg_entry_price"`
	CurrentPrice      string `json:"current_price"`
	MarketValue       string `json:"market_value"`
	CostBasis         string `json:"cost_basis"`
	UnrealizedPL      string `json:"unrealized_pl"`
	UnrealizedPLPC    string `json:"unrealized_plpc"`
	ChangeToday       string `json:"change_today"`
}

type providerAsset struct {
	ID           string `json:"id"`
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	Exchange     string `json:"exchange"`
	Class        string `json:"class"`
	Status       string `json:"status"`
	Tradable     bool   `json:"tradable"`
	Fractionable bool   `json:"fractionable"`
}

type providerOrder struct {
	ID            string    `json:"id"`
	ClientOrderID string    `json:"client_order_id"`
	Symbol        string    `json:"symbol"`
	Qty           string    `json:"qty"`
	FilledQty     string    `json:"filled_qty"`
	Side          string    `json:"side"`
	Type          string    `json:"type"`
	TimeInForce   string    `json:"time_in_force"`
	Status        string    `json:"status"`
	LimitPrice    *string   `json:"limit_price"`
	SubmittedAt   time.Time `json:"submitted_at"`
}

func normalizeAccount(value providerAccount) domain.TradingAccount {
	return domain.TradingAccount{
		ID: value.ID, Status: value.Status, Currency: value.Currency,
		Cash: decimal(value.Cash), BuyingPower: decimal(value.BuyingPower),
		PortfolioValue: decimal(value.PortfolioValue), Equity: decimal(value.Equity),
		LastEquity: decimal(value.LastEquity), LongMarketValue: decimal(value.LongMarketValue),
		TradingBlocked: value.TradingBlocked, PatternDayTrader: value.PatternDayTrader,
	}
}

func normalizePosition(value providerPosition) domain.Position {
	return domain.Position{
		Symbol: value.Symbol, AssetID: value.AssetID, Side: value.Side,
		Quantity: decimal(value.Qty), AverageEntryPrice: decimal(value.AverageEntryPrice),
		CurrentPrice: decimal(value.CurrentPrice), MarketValue: decimal(value.MarketValue),
		CostBasis: decimal(value.CostBasis), UnrealizedPL: decimal(value.UnrealizedPL),
		UnrealizedPLPercent: decimal(value.UnrealizedPLPC) * 100,
		ChangeToday:         decimal(value.ChangeToday) * 100,
	}
}

func normalizeAsset(value providerAsset) domain.TradingAsset {
	return domain.TradingAsset{
		ID: value.ID, Symbol: value.Symbol, Name: value.Name, Exchange: value.Exchange,
		AssetClass: value.Class, Status: value.Status, Tradable: value.Tradable,
		Fractionable: value.Fractionable,
	}
}

func normalizeOrder(value providerOrder) domain.Order {
	limitPrice := 0.0
	if value.LimitPrice != nil {
		limitPrice = decimal(*value.LimitPrice)
	}
	return domain.Order{
		ID: value.ID, ClientOrderID: value.ClientOrderID, Symbol: value.Symbol,
		Quantity: decimal(value.Qty), FilledQuantity: decimal(value.FilledQty), Side: value.Side,
		Type: value.Type, TimeInForce: value.TimeInForce, Status: value.Status,
		LimitPrice: limitPrice, SubmittedAt: value.SubmittedAt, Mode: "paper",
	}
}

func decimal(value string) float64 {
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}

func decimalString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func clonePortfolio(source domain.Portfolio) domain.Portfolio {
	cloned := source
	cloned.Positions = append([]domain.Position(nil), source.Positions...)
	return cloned
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
