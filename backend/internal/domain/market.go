package domain

import "time"

type MarketSymbol struct {
	Symbol string
	Name   string
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
