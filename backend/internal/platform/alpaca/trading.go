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

func (c *TradingClient) OrderMonitor(ctx context.Context) (domain.OrderMonitor, error) {
	ordersQuery := url.Values{
		"status":    {"all"},
		"limit":     {"500"},
		"direction": {"desc"},
		"nested":    {"true"},
	}
	var providerOrders []providerOrder
	if err := c.get(ctx, "/v2/orders", ordersQuery, &providerOrders); err != nil {
		return domain.OrderMonitor{}, err
	}
	fillsQuery := url.Values{
		"activity_types": {"FILL"},
		"direction":      {"desc"},
		"page_size":      {"100"},
	}
	var providerFills []providerFill
	if err := c.get(ctx, "/v2/account/activities", fillsQuery, &providerFills); err != nil {
		return domain.OrderMonitor{}, err
	}

	orders := make([]domain.Order, 0, len(providerOrders))
	workingCount := 0
	for _, value := range providerOrders {
		order := normalizeOrder(value)
		if order.Working {
			workingCount++
		}
		orders = append(orders, order)
	}
	fills := make([]domain.Fill, 0, len(providerFills))
	for _, value := range providerFills {
		fills = append(fills, normalizeFill(value))
	}
	return domain.OrderMonitor{
		Orders:       orders,
		Fills:        fills,
		AuditTrail:   buildAuditTrail(orders, fills),
		AsOf:         time.Now().UTC(),
		Source:       "alpaca",
		Mode:         "paper",
		OrderCount:   len(orders),
		WorkingCount: workingCount,
		FillCount:    len(fills),
	}, nil
}

func (c *TradingClient) CancelOrder(ctx context.Context, orderID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return &TradingAPIError{Status: http.StatusUnprocessableEntity, Message: "Order ID is required"}
	}
	if err := c.send(ctx, http.MethodDelete, "/v2/orders/"+url.PathEscape(orderID), nil, nil); err != nil {
		return err
	}
	c.portfolioCached = time.Time{}
	return nil
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
	if destination == nil || len(responseBody) == 0 {
		return nil
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
	ID                 string     `json:"id"`
	ClientOrderID      string     `json:"client_order_id"`
	Symbol             string     `json:"symbol"`
	Qty                string     `json:"qty"`
	FilledQty          string     `json:"filled_qty"`
	FilledAveragePrice *string    `json:"filled_avg_price"`
	Side               string     `json:"side"`
	Type               string     `json:"type"`
	TimeInForce        string     `json:"time_in_force"`
	Status             string     `json:"status"`
	LimitPrice         *string    `json:"limit_price"`
	SubmittedAt        time.Time  `json:"submitted_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	FilledAt           *time.Time `json:"filled_at"`
	CanceledAt         *time.Time `json:"canceled_at"`
	ExpiredAt          *time.Time `json:"expired_at"`
	FailedAt           *time.Time `json:"failed_at"`
}

type providerFill struct {
	ID              string    `json:"id"`
	OrderID         string    `json:"order_id"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"`
	Qty             string    `json:"qty"`
	CumulativeQty   string    `json:"cum_qty"`
	LeavesQty       string    `json:"leaves_qty"`
	Price           string    `json:"price"`
	Type            string    `json:"type"`
	TransactionTime time.Time `json:"transaction_time"`
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
	filledAveragePrice := 0.0
	if value.FilledAveragePrice != nil {
		filledAveragePrice = decimal(*value.FilledAveragePrice)
	}
	updatedAt := value.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = value.SubmittedAt
	}
	return domain.Order{
		ID: value.ID, ClientOrderID: value.ClientOrderID, Symbol: value.Symbol,
		Quantity: decimal(value.Qty), FilledQuantity: decimal(value.FilledQty), Side: value.Side,
		Type: value.Type, TimeInForce: value.TimeInForce, Status: value.Status,
		LimitPrice: limitPrice, FilledAveragePrice: filledAveragePrice,
		SubmittedAt: value.SubmittedAt, UpdatedAt: updatedAt, FilledAt: value.FilledAt,
		CanceledAt: value.CanceledAt, ExpiredAt: value.ExpiredAt, FailedAt: value.FailedAt,
		Working: workingOrderStatus(value.Status), Mode: "paper",
	}
}

func normalizeFill(value providerFill) domain.Fill {
	return domain.Fill{
		ID: value.ID, OrderID: value.OrderID, Symbol: value.Symbol, Side: value.Side,
		Quantity: decimal(value.Qty), CumulativeQty: decimal(value.CumulativeQty),
		LeavesQty: decimal(value.LeavesQty), Price: decimal(value.Price), Type: value.Type,
		TransactionTime: value.TransactionTime, Source: "alpaca", Mode: "paper",
	}
}

func workingOrderStatus(status string) bool {
	switch status {
	case "new", "accepted", "pending_new", "accepted_for_bidding", "partially_filled", "pending_replace", "calculated":
		return true
	default:
		return false
	}
}

func buildAuditTrail(orders []domain.Order, fills []domain.Fill) []domain.OrderAuditEvent {
	events := make([]domain.OrderAuditEvent, 0, len(orders)*2+len(fills))
	for _, order := range orders {
		events = append(events, domain.OrderAuditEvent{
			ID: order.ID + ":submitted", OrderID: order.ID, Symbol: order.Symbol,
			Timestamp: order.SubmittedAt, Event: "SUBMITTED", Status: "submitted", Side: order.Side,
			Quantity: order.Quantity, FilledQuantity: 0, Price: order.LimitPrice,
			Message: fmt.Sprintf("%s %s order submitted to Alpaca paper trading", strings.ToUpper(order.Side), order.Type),
			Source:  "alpaca_order_api",
		})
		terminalTime, eventName := orderTerminalEvent(order)
		if terminalTime != nil {
			events = append(events, domain.OrderAuditEvent{
				ID: order.ID + ":" + strings.ToLower(eventName), OrderID: order.ID, Symbol: order.Symbol,
				Timestamp: *terminalTime, Event: eventName, Status: order.Status, Side: order.Side,
				Quantity: order.Quantity, FilledQuantity: order.FilledQuantity,
				Price: order.FilledAveragePrice, Message: orderAuditMessage(order, eventName),
				Source: "alpaca_order_api",
			})
		} else if order.UpdatedAt.After(order.SubmittedAt) {
			events = append(events, domain.OrderAuditEvent{
				ID: order.ID + ":status", OrderID: order.ID, Symbol: order.Symbol,
				Timestamp: order.UpdatedAt, Event: "STATUS", Status: order.Status, Side: order.Side,
				Quantity: order.Quantity, FilledQuantity: order.FilledQuantity,
				Price: order.LimitPrice, Message: "Order status updated to " + order.Status,
				Source: "alpaca_order_api",
			})
		}
	}
	for _, fill := range fills {
		eventName := "PARTIAL_FILL"
		if fill.LeavesQty == 0 {
			eventName = "FILL"
		}
		events = append(events, domain.OrderAuditEvent{
			ID: fill.ID, OrderID: fill.OrderID, Symbol: fill.Symbol, Timestamp: fill.TransactionTime,
			Event: eventName, Status: fill.Type, Side: fill.Side, Quantity: fill.Quantity,
			FilledQuantity: fill.CumulativeQty, Price: fill.Price,
			Message: fmt.Sprintf("Executed %s %s at %s", decimalString(fill.Quantity), fill.Symbol, decimalString(fill.Price)),
			Source:  "alpaca_account_activity",
		})
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Timestamp.After(events[j].Timestamp) })
	return events
}

func orderTerminalEvent(order domain.Order) (*time.Time, string) {
	if order.FailedAt != nil {
		return order.FailedAt, "REJECTED"
	}
	if order.CanceledAt != nil {
		return order.CanceledAt, "CANCELED"
	}
	if order.ExpiredAt != nil {
		return order.ExpiredAt, "EXPIRED"
	}
	if order.FilledAt != nil {
		return order.FilledAt, "FILLED"
	}
	return nil, ""
}

func orderAuditMessage(order domain.Order, event string) string {
	switch event {
	case "FILLED":
		return fmt.Sprintf("Order fully filled: %s at average %s", decimalString(order.FilledQuantity), decimalString(order.FilledAveragePrice))
	case "CANCELED":
		return "Remaining working quantity canceled"
	case "REJECTED":
		return "Order rejected by the paper-trading venue"
	case "EXPIRED":
		return "Order expired before the remaining quantity filled"
	default:
		return "Order status updated"
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
