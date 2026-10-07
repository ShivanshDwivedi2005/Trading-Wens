package domain

import (
	"errors"
	"time"
)

var (
	ErrUnsupportedSymbol = errors.New("unsupported market symbol")
	ErrUnsupportedRange  = errors.New("unsupported market history range")
)

type MarketSymbol struct {
	Symbol  string
	Name    string
	Aliases []string
}

type MarketSnapshot struct {
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	Open          float64   `json:"open"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	PreviousClose float64   `json:"previous_close"`
	Volume        uint64    `json:"volume"`
	Timestamp     time.Time `json:"timestamp"`
	Available     bool      `json:"available"`
}

type MarketSnapshotSet struct {
	Data   []MarketSnapshot `json:"data"`
	AsOf   time.Time        `json:"as_of"`
	Feed   string           `json:"feed"`
	Source string           `json:"source"`
	Count  int              `json:"count"`
}

type MarketBar struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    uint64    `json:"volume"`
}

type StockHistory struct {
	Symbol    string      `json:"symbol"`
	Name      string      `json:"name"`
	Data      []MarketBar `json:"data"`
	AsOf      time.Time   `json:"as_of"`
	Range     string      `json:"range"`
	Timeframe string      `json:"timeframe"`
	Feed      string      `json:"feed"`
	Source    string      `json:"source"`
	Count     int         `json:"count"`
}
