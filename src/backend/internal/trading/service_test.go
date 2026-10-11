package trading

import (
	"context"
	"testing"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

type providerStub struct {
	portfolio domain.Portfolio
	monitor   domain.OrderMonitor
}

func (s providerStub) Portfolio(context.Context) (domain.Portfolio, error) { return s.portfolio, nil }
func (s providerStub) Assets(context.Context, string) ([]domain.TradingAsset, error) {
	return nil, nil
}
func (s providerStub) SubmitOrder(context.Context, domain.OrderRequest) (domain.Order, error) {
	return domain.Order{}, nil
}
func (s providerStub) OrderMonitor(context.Context) (domain.OrderMonitor, error) {
	return s.monitor, nil
}
func (s providerStub) CancelOrder(context.Context, string) error { return nil }

type ledgerStub struct {
	user      domain.User
	portfolio domain.Portfolio
	monitor   domain.OrderMonitor
	reads     int
}

func (s *ledgerStub) SyncTradingState(
	_ context.Context,
	user domain.User,
	portfolio domain.Portfolio,
	monitor domain.OrderMonitor,
) error {
	s.user = user
	s.portfolio = portfolio
	s.monitor = monitor
	return nil
}

func (s *ledgerStub) TradingState(
	_ context.Context,
	_ domain.User,
	_, _ string,
) (domain.Portfolio, domain.OrderMonitor, error) {
	s.reads++
	return s.portfolio, s.monitor, nil
}

type newsStub struct{ feed domain.NewsFeed }

func (s newsStub) Latest(context.Context) (domain.NewsFeed, error) { return s.feed, nil }

type refreshingProviderStub struct {
	providerStub
	portfolioCalls int
}

func (s *refreshingProviderStub) Portfolio(context.Context) (domain.Portfolio, error) {
	s.portfolioCalls++
	portfolio := s.portfolio
	if s.portfolioCalls > 1 {
		portfolio.Positions = []domain.Position{{Symbol: "MSFT", Quantity: 2, MarketValue: 800}}
	}
	return portfolio, nil
}

func (s *refreshingProviderStub) SubmitOrder(context.Context, domain.OrderRequest) (domain.Order, error) {
	return domain.Order{ID: "order-1", Symbol: "MSFT", Status: "filled"}, nil
}

func TestSubmitOrderPersistsPostOrderPortfolio(t *testing.T) {
	now := time.Now().UTC()
	provider := &refreshingProviderStub{providerStub: providerStub{
		portfolio: domain.Portfolio{
			Account: domain.TradingAccount{ID: "account-1"}, AsOf: now, Source: "alpaca", Mode: "paper",
		},
		monitor: domain.OrderMonitor{
			Orders: []domain.Order{{ID: "order-1", Symbol: "MSFT", Status: "filled"}},
			AsOf:   now, Source: "alpaca", Mode: "paper",
		},
	}}
	ledger := &ledgerStub{}
	service := NewService(provider, ledger, nil)
	ctx := domain.WithAuthenticatedUser(context.Background(), domain.User{ID: "user-1"})

	if _, err := service.SubmitOrder(ctx, domain.OrderRequest{Symbol: "MSFT", Quantity: 2, Side: "buy", Type: "market"}); err != nil {
		t.Fatal(err)
	}
	if provider.portfolioCalls < 2 || len(ledger.portfolio.Positions) != 1 || ledger.portfolio.Positions[0].Symbol != "MSFT" {
		t.Fatalf("post-order portfolio was not persisted: calls=%d portfolio=%#v", provider.portfolioCalls, ledger.portfolio)
	}
}

func TestOrderMonitorPersistsTenantScopedSentimentSignal(t *testing.T) {
	now := time.Now().UTC()
	provider := providerStub{
		portfolio: domain.Portfolio{
			Account: domain.TradingAccount{ID: "account-1"}, AsOf: now, Source: "alpaca", Mode: "paper",
		},
		monitor: domain.OrderMonitor{
			Orders: []domain.Order{{ID: "order-1", Symbol: "AAPL", Working: true}},
			AsOf:   now, Source: "alpaca", Mode: "paper",
		},
	}
	ledger := &ledgerStub{}
	news := newsStub{feed: domain.NewsFeed{Data: []domain.NewsArticle{{
		Provider: "gdelt", MatchedSymbols: []string{"AAPL"},
		Sentiment: &domain.SentimentAnalysis{Label: "POSITIVE", Score: 0.6, Confidence: 0.8, Model: "finbert"},
	}}}}
	service := NewService(provider, ledger, news)
	ctx := domain.WithAuthenticatedUser(context.Background(), domain.User{ID: "user-1", Email: "a@example.com"})

	monitor, err := service.OrderMonitor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	signal := monitor.Orders[0].Signal
	if signal == nil || signal.PriceDirection != "UP" || signal.Sentiment != "POSITIVE" || signal.Possibility != 74 {
		t.Fatalf("unexpected signal: %#v", signal)
	}
	if ledger.user.ID != "user-1" || ledger.monitor.Orders[0].Signal == nil {
		t.Fatalf("state was not persisted for the authenticated user: %#v", ledger)
	}
	if ledger.reads != 1 {
		t.Fatalf("expected response to be loaded from the database, got %d reads", ledger.reads)
	}
}

func TestOrderMonitorRequiresAuthenticatedUser(t *testing.T) {
	service := NewService(providerStub{}, &ledgerStub{}, nil)
	if _, err := service.OrderMonitor(context.Background()); err != ErrUnauthenticated {
		t.Fatalf("expected unauthenticated error, got %v", err)
	}
}

func TestPotentialWashTradeAllowsSeparatedLimitOrders(t *testing.T) {
	workingSell := domain.Order{
		ID: "sell-1", Symbol: "AAPL", Side: "sell", Type: "limit", LimitPrice: 220, Working: true,
	}
	request := domain.OrderRequest{
		Symbol: "AAPL", Side: "buy", Type: "limit", LimitPrice: 210,
	}

	if conflict := potentialWashTrade(request, []domain.Order{workingSell}); conflict != nil {
		t.Fatalf("buy limit below sell limit must remain valid, got conflict %#v", conflict)
	}
}

func TestPotentialWashTradeRejectsCrossedOrMarketOrders(t *testing.T) {
	workingSell := domain.Order{
		ID: "sell-1", Symbol: "AAPL", Side: "sell", Type: "limit", LimitPrice: 220, Working: true,
	}
	tests := []domain.OrderRequest{
		{Symbol: "AAPL", Side: "buy", Type: "limit", LimitPrice: 220},
		{Symbol: "AAPL", Side: "buy", Type: "limit", LimitPrice: 225},
		{Symbol: "AAPL", Side: "buy", Type: "market"},
	}

	for _, request := range tests {
		if conflict := potentialWashTrade(request, []domain.Order{workingSell}); conflict == nil {
			t.Fatalf("expected conflict for %#v", request)
		}
	}
}
