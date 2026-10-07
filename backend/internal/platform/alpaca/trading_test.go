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
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode order: %v", err)
			}
			if payload["symbol"] != "AAPL" || payload["qty"] != "2" || payload["limit_price"] != "215.5" {
				t.Fatalf("unexpected order payload: %#v", payload)
			}
			_, _ = w.Write([]byte(`{"id":"order-1","client_order_id":"client-1","symbol":"AAPL","qty":"2","filled_qty":"0","side":"buy","type":"limit","time_in_force":"day","status":"accepted","limit_price":"215.5","submitted_at":"2026-10-08T10:00:00Z"}`))
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
}
