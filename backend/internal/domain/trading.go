package domain

import "time"

type TradingAccount struct {
	ID               string  `json:"id"`
	Status           string  `json:"status"`
	Currency         string  `json:"currency"`
	Cash             float64 `json:"cash"`
	BuyingPower      float64 `json:"buying_power"`
	PortfolioValue   float64 `json:"portfolio_value"`
	Equity           float64 `json:"equity"`
	LastEquity       float64 `json:"last_equity"`
	LongMarketValue  float64 `json:"long_market_value"`
	TradingBlocked   bool    `json:"trading_blocked"`
	PatternDayTrader bool    `json:"pattern_day_trader"`
}

type Position struct {
	Symbol              string  `json:"symbol"`
	AssetID             string  `json:"asset_id"`
	Side                string  `json:"side"`
	Quantity            float64 `json:"quantity"`
	AverageEntryPrice   float64 `json:"average_entry_price"`
	CurrentPrice        float64 `json:"current_price"`
	MarketValue         float64 `json:"market_value"`
	CostBasis           float64 `json:"cost_basis"`
	UnrealizedPL        float64 `json:"unrealized_pl"`
	UnrealizedPLPercent float64 `json:"unrealized_pl_percent"`
	ChangeToday         float64 `json:"change_today"`
}

type Portfolio struct {
	Account   TradingAccount `json:"account"`
	Positions []Position     `json:"positions"`
	AsOf      time.Time      `json:"as_of"`
	Source    string         `json:"source"`
	Mode      string         `json:"mode"`
}

type TradingAsset struct {
	ID           string `json:"id"`
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	Exchange     string `json:"exchange"`
	AssetClass   string `json:"asset_class"`
	Status       string `json:"status"`
	Tradable     bool   `json:"tradable"`
	Fractionable bool   `json:"fractionable"`
}

type OrderRequest struct {
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity"`
	Side        string  `json:"side"`
	Type        string  `json:"type"`
	LimitPrice  float64 `json:"limit_price,omitempty"`
	TimeInForce string  `json:"time_in_force"`
}

type Order struct {
	ID             string    `json:"id"`
	ClientOrderID  string    `json:"client_order_id"`
	Symbol         string    `json:"symbol"`
	Quantity       float64   `json:"quantity"`
	FilledQuantity float64   `json:"filled_quantity"`
	Side           string    `json:"side"`
	Type           string    `json:"type"`
	TimeInForce    string    `json:"time_in_force"`
	Status         string    `json:"status"`
	LimitPrice     float64   `json:"limit_price,omitempty"`
	SubmittedAt    time.Time `json:"submitted_at"`
	Mode           string    `json:"mode"`
}
