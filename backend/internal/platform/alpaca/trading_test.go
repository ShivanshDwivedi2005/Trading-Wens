package alpaca

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

func TestTradingClientNormalizesPortfolioAssetsAndPaperOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/account":
			_, _ = w.Write([]byte(`{"id":"account-1","status":"ACTIVE","currency":"USD","cash":"25000","buying_power":"50000","portfolio_value":"102500","equity":"102500","last_equity":"100000","long_market_value":"77500","trading_blocked":false}`))
		case "/v2/positions":
			_, _ = w.Write([]byte(`[{"symbol":"AAPL","asset_id":"asset-1","side":"long","qty":"10","avg_entry_price":"200","current_price":"220","market_value":"2200","cost_basis":"2000","unrealized_pl":"200","unrealized_plpc":"0.1","change_today":"0.025"}]`))
		case "/v2/assets":
			_, _ = w.Write([]byte(`[{"id":"asset-1","symbol":"AAPL","name":"Apple Inc.","exchange":"NASDAQ","class":"us_equity","status":"active","tradable":true,"fractionable":true}]`))
		case "/v2/assets/AAPL":
			_, _ = w.Write([]byte(`{"id":"asset-1","symbol":"AAPL","name":"Apple Inc.","exchange":"NASDAQ","class":"us_equity","status":"active","tradable":true,"fractionable":true}`))
		case "/v2/orders":
			if r.Method == http.MethodGet {
				if r.URL.Query().Get("status") != "all" || r.URL.Query().Get("limit") != "500" {
					t.Fatalf("unexpected order query: %s", r.URL.RawQuery)
				}
				_, _ = w.Write([]byte(`[{"id":"order-1","client_order_id":"client-1","symbol":"AAPL","qty":"2","filled_qty":"1","filled_avg_price":"214.5","side":"buy","type":"limit","time_in_force":"day","status":"partially_filled","limit_price":"215.5","submitted_at":"2026-10-08T10:00:00Z","updated_at":"2026-10-08T10:01:00Z"}]`))
				return
			}
			if r.Method != http.MethodPost {
				t.Fatalf("expected GET or POST, got %s", r.Method)
			}
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode order: %v", err)
			}
			if payload["symbol"] != "AAPL" || payload["qty"] != "2" || payload["limit_price"] != "215.5" {
				t.Fatalf("unexpected order payload: %#v", payload)
			}
			_, _ = w.Write([]byte(`{"id":"order-1","client_order_id":"client-1","symbol":"AAPL","qty":"2","filled_qty":"0","side":"buy","type":"limit","time_in_force":"day","status":"accepted","limit_price":"215.5","submitted_at":"2026-10-08T10:00:00Z"}`))
		case "/v2/account/activities":
			if r.URL.Query().Get("activity_types") != "FILL" || r.URL.Query().Get("page_size") != "100" {
				t.Fatalf("unexpected activities query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[{"id":"fill-1","order_id":"order-1","symbol":"AAPL","side":"buy","qty":"1","cum_qty":"1","leaves_qty":"1","price":"214.5","type":"partial_fill","transaction_time":"2026-10-08T10:02:00Z"}]`))
		case "/v2/orders/order-1":
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewTradingClient(server.URL, "key", "secret", server.Client())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	portfolio, err := client.Portfolio(context.Background())
	if err != nil {
		t.Fatalf("portfolio: %v", err)
	}
	if portfolio.Mode != "paper" || portfolio.Account.PortfolioValue != 102500 || portfolio.Positions[0].ChangeToday != 2.5 {
		t.Fatalf("unexpected portfolio: %#v", portfolio)
	}
	assets, err := client.Assets(context.Background(), "apple")
	if err != nil || len(assets) != 1 || assets[0].Symbol != "AAPL" {
		t.Fatalf("unexpected assets: %#v err=%v", assets, err)
	}
	order, err := client.SubmitOrder(context.Background(), domain.OrderRequest{
		Symbol: "AAPL", Quantity: 2, Side: "buy", Type: "limit", LimitPrice: 215.5, TimeInForce: "day",
	})
	if err != nil {
		t.Fatalf("submit order: %v", err)
	}
	if order.Mode != "paper" || order.ID != "order-1" || order.LimitPrice != 215.5 {
		t.Fatalf("unexpected order: %#v", order)
	}
	monitor, err := client.OrderMonitor(context.Background())
	if err != nil {
		t.Fatalf("order monitor: %v", err)
	}
	if monitor.WorkingCount != 1 || monitor.FillCount != 1 || len(monitor.AuditTrail) != 3 {
		t.Fatalf("unexpected order monitor: %#v", monitor)
	}
	if monitor.Fills[0].Price != 214.5 || monitor.AuditTrail[0].Event != "PARTIAL_FILL" {
		t.Fatalf("unexpected execution data: %#v", monitor)
	}
	if err := client.CancelOrder(context.Background(), "order-1"); err != nil {
		t.Fatalf("cancel order: %v", err)
	}
}
