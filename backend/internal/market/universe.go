package market

import "github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"

// SP500TopThirty is the maintained large-cap universe shown in the market overview.
// It is intentionally kept server-side so provider symbols and display names have one owner.
var SP500TopThirty = []domain.MarketSymbol{
	{Symbol: "NVDA", Name: "NVIDIA"},
	{Symbol: "AAPL", Name: "Apple"},
	{Symbol: "GOOGL", Name: "Alphabet", Aliases: []string{"Google"}},
	{Symbol: "MSFT", Name: "Microsoft"},
	{Symbol: "AMZN", Name: "Amazon"},
	{Symbol: "AVGO", Name: "Broadcom"},
	{Symbol: "META", Name: "Meta Platforms", Aliases: []string{"Facebook"}},
	{Symbol: "TSLA", Name: "Tesla"},
	{Symbol: "BRK.B", Name: "Berkshire Hathaway"},
	{Symbol: "LLY", Name: "Eli Lilly"},
	{Symbol: "JPM", Name: "JPMorgan Chase"},
	{Symbol: "WMT", Name: "Walmart"},
	{Symbol: "V", Name: "Visa"},
	{Symbol: "ORCL", Name: "Oracle"},
	{Symbol: "XOM", Name: "Exxon Mobil"},
	{Symbol: "JNJ", Name: "Johnson & Johnson", Aliases: []string{"Johnson and Johnson"}},
	{Symbol: "MA", Name: "Mastercard"},
	{Symbol: "NFLX", Name: "Netflix"},
	{Symbol: "COST", Name: "Costco"},
	{Symbol: "ABBV", Name: "AbbVie"},
	{Symbol: "HD", Name: "Home Depot"},
	{Symbol: "PG", Name: "Procter & Gamble", Aliases: []string{"Procter and Gamble"}},
	{Symbol: "BAC", Name: "Bank of America"},
	{Symbol: "GE", Name: "GE Aerospace"},
	{Symbol: "KO", Name: "Coca-Cola", Aliases: []string{"Coca Cola"}},
	{Symbol: "CSCO", Name: "Cisco"},
	{Symbol: "CAT", Name: "Caterpillar"},
	{Symbol: "PM", Name: "Philip Morris"},
	{Symbol: "IBM", Name: "IBM"},
	{Symbol: "CVX", Name: "Chevron"},
}
