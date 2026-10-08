package trading

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

var ErrUnauthenticated = errors.New("authenticated user is required")

type Provider interface {
	Portfolio(ctx context.Context) (domain.Portfolio, error)
	Assets(ctx context.Context, search string) ([]domain.TradingAsset, error)
	SubmitOrder(ctx context.Context, request domain.OrderRequest) (domain.Order, error)
	OrderMonitor(ctx context.Context) (domain.OrderMonitor, error)
	CancelOrder(ctx context.Context, orderID string) error
}

type Ledger interface {
	SyncTradingState(
		ctx context.Context,
		user domain.User,
		portfolio domain.Portfolio,
		monitor domain.OrderMonitor,
	) error
	TradingState(
		ctx context.Context,
		user domain.User,
		provider, accountID string,
	) (domain.Portfolio, domain.OrderMonitor, error)
}

type NewsService interface {
	Latest(ctx context.Context) (domain.NewsFeed, error)
}

type Service struct {
	provider Provider
	ledger   Ledger
	news     NewsService

	signalMu       sync.Mutex
	signalFeed     domain.NewsFeed
	signalCachedAt time.Time
}

func NewService(provider Provider, ledger Ledger, news NewsService) *Service {
	return &Service{provider: provider, ledger: ledger, news: news}
}

func (s *Service) Portfolio(ctx context.Context) (domain.Portfolio, error) {
	user, err := authenticatedUser(ctx)
	if err != nil {
		return domain.Portfolio{}, err
	}
	portfolio, err := s.provider.Portfolio(ctx)
	if err != nil {
		return domain.Portfolio{}, err
	}
	if err := s.ledger.SyncTradingState(ctx, user, portfolio, domain.OrderMonitor{}); err != nil {
		return domain.Portfolio{}, fmt.Errorf("persist portfolio: %w", err)
	}
	stored, _, err := s.ledger.TradingState(ctx, user, portfolio.Source, portfolio.Account.ID)
	if err != nil {
		return domain.Portfolio{}, fmt.Errorf("load persisted portfolio: %w", err)
	}
	return stored, nil
}

func (s *Service) Assets(ctx context.Context, search string) ([]domain.TradingAsset, error) {
	if _, err := authenticatedUser(ctx); err != nil {
		return nil, err
	}
	return s.provider.Assets(ctx, search)
}

func (s *Service) SubmitOrder(ctx context.Context, request domain.OrderRequest) (domain.Order, error) {
	user, err := authenticatedUser(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	portfolio, err := s.provider.Portfolio(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	order, err := s.provider.SubmitOrder(ctx, request)
	if err != nil {
		return domain.Order{}, err
	}
	monitor, monitorErr := s.provider.OrderMonitor(ctx)
	if monitorErr != nil {
		monitor = monitorForSubmittedOrder(order)
	}
	s.enrichSignals(ctx, &monitor)
	if err := s.ledger.SyncTradingState(ctx, user, portfolio, monitor); err != nil {
		return domain.Order{}, fmt.Errorf("persist submitted order: %w", err)
	}
	_, storedMonitor, err := s.ledger.TradingState(ctx, user, portfolio.Source, portfolio.Account.ID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("load persisted submitted order: %w", err)
	}
	for _, persisted := range storedMonitor.Orders {
		if persisted.ID == order.ID {
			return persisted, nil
		}
	}
	return order, nil
}

func (s *Service) OrderMonitor(ctx context.Context) (domain.OrderMonitor, error) {
	user, err := authenticatedUser(ctx)
	if err != nil {
		return domain.OrderMonitor{}, err
	}
	portfolio, err := s.provider.Portfolio(ctx)
	if err != nil {
		return domain.OrderMonitor{}, err
	}
	monitor, err := s.provider.OrderMonitor(ctx)
	if err != nil {
		return domain.OrderMonitor{}, err
	}
	s.enrichSignals(ctx, &monitor)
	if err := s.ledger.SyncTradingState(ctx, user, portfolio, monitor); err != nil {
		return domain.OrderMonitor{}, fmt.Errorf("persist order activity: %w", err)
	}
	_, stored, err := s.ledger.TradingState(ctx, user, portfolio.Source, portfolio.Account.ID)
	if err != nil {
		return domain.OrderMonitor{}, fmt.Errorf("load persisted order activity: %w", err)
	}
	return stored, nil
}

func (s *Service) CancelOrder(ctx context.Context, orderID string) error {
	user, err := authenticatedUser(ctx)
	if err != nil {
		return err
	}
	portfolio, err := s.provider.Portfolio(ctx)
	if err != nil {
		return err
	}
	monitor, err := s.provider.OrderMonitor(ctx)
	if err != nil {
		return err
	}
	if !containsWorkingOrder(monitor.Orders, orderID) {
		return errors.New("working order does not belong to the authenticated account")
	}
	if err := s.ledger.SyncTradingState(ctx, user, portfolio, monitor); err != nil {
		return fmt.Errorf("persist order ownership: %w", err)
	}
	if err := s.provider.CancelOrder(ctx, orderID); err != nil {
		return err
	}
	updated, err := s.provider.OrderMonitor(ctx)
	if err == nil {
		s.enrichSignals(ctx, &updated)
		_ = s.ledger.SyncTradingState(ctx, user, portfolio, updated)
	}
	return nil
}

func (s *Service) enrichSignals(ctx context.Context, monitor *domain.OrderMonitor) {
	if s.news == nil || len(monitor.Orders) == 0 {
		return
	}
	feed, err := s.latestSignalFeed(ctx)
	if err != nil {
		return
	}
	for index := range monitor.Orders {
		monitor.Orders[index].Signal = signalForSymbol(monitor.Orders[index].Symbol, feed.Data)
	}
}

func (s *Service) latestSignalFeed(ctx context.Context) (domain.NewsFeed, error) {
	s.signalMu.Lock()
	defer s.signalMu.Unlock()
	if !s.signalCachedAt.IsZero() && time.Since(s.signalCachedAt) < 2*time.Minute {
		return s.signalFeed, nil
	}
	feed, err := s.news.Latest(ctx)
	if err != nil {
		return domain.NewsFeed{}, err
	}
	s.signalFeed = feed
	s.signalCachedAt = time.Now()
	return feed, nil
}

func signalForSymbol(symbol string, articles []domain.NewsArticle) *domain.OrderSignal {
	var weightedScore, totalWeight, confidence float64
	providers := make(map[string]struct{})
	models := make(map[string]struct{})
	count := 0
	for _, article := range articles {
		if article.Sentiment == nil || !containsSymbol(article.MatchedSymbols, symbol) {
			continue
		}
		weight := article.Sentiment.Confidence
		if weight <= 0 {
			weight = 0.1
		}
		weightedScore += article.Sentiment.Score * weight
		totalWeight += weight
		confidence += article.Sentiment.Confidence
		count++
		if article.Provider != "" {
			providers[article.Provider] = struct{}{}
		}
		if article.Sentiment.Model != "" {
			models[article.Sentiment.Model] = struct{}{}
		}
	}
	if count == 0 {
		return &domain.OrderSignal{
			Sentiment: "NEUTRAL", PriceDirection: "SIDEWAYS", Possibility: 50,
			Reason:  "No current model-scored headline matched this symbol; sentiment contributes a neutral direction.",
			Sources: []string{"No matched sentiment evidence"},
		}
	}
	score := weightedScore / totalWeight
	averageConfidence := confidence / float64(count)
	label, direction := "NEUTRAL", "SIDEWAYS"
	if score > 0.08 {
		label, direction = "POSITIVE", "UP"
	} else if score < -0.08 {
		label, direction = "NEGATIVE", "DOWN"
	}
	possibility := int(math.Round(50 + math.Min(math.Abs(score), 1)*40))
	sources := make([]string, 0, len(providers)+len(models))
	for provider := range providers {
		sources = append(sources, provider+" news")
	}
	for model := range models {
		sources = append(sources, model+" sentiment")
	}
	sort.Strings(sources)
	return &domain.OrderSignal{
		Sentiment: label, SentimentScore: round(score, 4), Confidence: round(averageConfidence, 4),
		PriceDirection: direction, Possibility: possibility, AnalyzedHeadlines: count,
		Reason: fmt.Sprintf(
			"%d model-scored headline(s) average %+.2f sentiment; this supports a %s price bias, not a guaranteed move.",
			count, score, strings.ToLower(direction),
		),
		Sources: sources,
	}
}

func monitorForSubmittedOrder(order domain.Order) domain.OrderMonitor {
	return domain.OrderMonitor{
		Orders: []domain.Order{order},
		AuditTrail: []domain.OrderAuditEvent{{
			ID: order.ID + ":submitted", OrderID: order.ID, Symbol: order.Symbol,
			Timestamp: order.SubmittedAt, Event: "SUBMITTED", Status: order.Status,
			Side: order.Side, Quantity: order.Quantity, FilledQuantity: order.FilledQuantity,
			Price: order.LimitPrice, Message: "Paper order accepted by Alpaca", Source: "alpaca",
		}},
		AsOf: order.UpdatedAt, Source: "alpaca", Mode: "paper", OrderCount: 1,
	}
}

func authenticatedUser(ctx context.Context) (domain.User, error) {
	user, ok := domain.AuthenticatedUser(ctx)
	if !ok {
		return domain.User{}, ErrUnauthenticated
	}
	return user, nil
}

func containsSymbol(symbols []string, wanted string) bool {
	for _, symbol := range symbols {
		if strings.EqualFold(strings.TrimSpace(symbol), strings.TrimSpace(wanted)) {
			return true
		}
	}
	return false
}

func containsWorkingOrder(orders []domain.Order, orderID string) bool {
	for _, order := range orders {
		if order.ID == orderID && order.Working {
			return true
		}
	}
	return false
}

func round(value float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(value*factor) / factor
}
