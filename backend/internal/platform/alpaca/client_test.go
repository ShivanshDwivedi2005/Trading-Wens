package alpaca

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

func TestSnapshotsNormalizesAndCachesProviderData(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v2/stocks/snapshots" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("symbols") != "AAPL,MSFT" || r.URL.Query().Get("feed") != "iex" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		if r.Header.Get("APCA-API-KEY-ID") != "key-id" || r.Header.Get("APCA-API-SECRET-KEY") != "secret-key" {
			t.Fatal("missing Alpaca credentials")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"AAPL": {
				"latestTrade": {"t":"2026-10-06T15:59:59Z","p":251.5},
				"dailyBar": {"t":"2026-10-06T13:30:00Z","o":248,"h":253,"l":247.5,"c":251.4,"v":42000000},
				"prevDailyBar": {"t":"2026-10-05T20:00:00Z","c":250}
			},
			"MSFT": {
				"minuteBar": {"t":"2026-10-06T15:59:00Z","c":510.25},
				"dailyBar": {"t":"2026-10-06T13:30:00Z","o":505,"h":512,"l":503,"c":510.25,"v":18000000},
				"prevDailyBar": {"t":"2026-10-05T20:00:00Z","c":500}
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(
		server.URL,
		"key-id",
		"secret-key",
		"iex",
		[]domain.MarketSymbol{{Symbol: "AAPL", Name: "Apple"}, {Symbol: "MSFT", Name: "Microsoft"}},
		server.Client(),
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	first, err := client.Snapshots(context.Background())
	if err != nil {
		t.Fatalf("fetch snapshots: %v", err)
	}
	second, err := client.Snapshots(context.Background())
	if err != nil {
		t.Fatalf("fetch cached snapshots: %v", err)
	}

	if requests != 1 {
		t.Fatalf("expected one provider request, got %d", requests)
	}
	if first.Count != 2 || first.Feed != "iex" || first.Source != "alpaca" || len(second.Data) != 2 {
		t.Fatalf("unexpected response metadata: %#v", first)
	}
	if first.Data[0].Symbol != "AAPL" || first.Data[0].Price != 251.5 || first.Data[0].ChangePercent != 0.6 {
		t.Fatalf("unexpected AAPL snapshot: %#v", first.Data[0])
	}
	if first.Data[1].Symbol != "MSFT" || first.Data[1].Price != 510.25 || !first.Data[1].Available {
		t.Fatalf("unexpected MSFT snapshot: %#v", first.Data[1])
	}
}

func TestSnapshotsRejectsProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewClient(
		server.URL,
		"key-id",
		"secret-key",
		"iex",
		[]domain.MarketSymbol{{Symbol: "AAPL", Name: "Apple"}},
		server.Client(),
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if _, err := client.Snapshots(context.Background()); err == nil {
		t.Fatal("expected provider error")
	}
}

func TestHistoryNormalizesAndCachesBars(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v2/stocks/AAPL/bars" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("timeframe") != "15Min" || r.URL.Query().Get("feed") != "iex" || r.URL.Query().Get("sort") != "asc" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bars":[
			{"t":"2026-10-06T15:30:00Z","o":250,"h":252,"l":249.5,"c":251.5,"v":125000},
			{"t":"2026-10-06T15:45:00Z","o":251.5,"h":253,"l":251,"c":252.75,"v":140000}
		]}`))
	}))
	defer server.Close()

	client, err := NewClient(
		server.URL,
		"key-id",
		"secret-key",
		"iex",
		[]domain.MarketSymbol{{Symbol: "AAPL", Name: "Apple"}},
		server.Client(),
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	first, err := client.History(context.Background(), "aapl", "5d")
	if err != nil {
		t.Fatalf("fetch history: %v", err)
	}
	second, err := client.History(context.Background(), "AAPL", "5D")
	if err != nil {
		t.Fatalf("fetch cached history: %v", err)
	}
	if requests != 1 || first.Count != 2 || len(second.Data) != 2 {
		t.Fatalf("unexpected history cache result: requests=%d result=%#v", requests, first)
	}
	if first.Symbol != "AAPL" || first.Range != "5D" || first.Timeframe != "15Min" || first.Data[1].Close != 252.75 {
		t.Fatalf("unexpected normalized history: %#v", first)
	}
}

func TestHistoryRejectsUnknownSymbolAndRange(t *testing.T) {
	client, err := NewClient(
		"https://data.example.com",
		"key-id",
		"secret-key",
		"iex",
		[]domain.MarketSymbol{{Symbol: "AAPL", Name: "Apple"}},
		nil,
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := client.History(context.Background(), "MSFT", "1D"); err != domain.ErrUnsupportedSymbol {
		t.Fatalf("expected unsupported symbol, got %v", err)
	}
	if _, err := client.History(context.Background(), "AAPL", "1Y"); err != domain.ErrUnsupportedRange {
		t.Fatalf("expected unsupported range, got %v", err)
	}
}
