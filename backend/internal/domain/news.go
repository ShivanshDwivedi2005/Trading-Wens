package domain

import "time"

type NewsArticle struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	Domain         string    `json:"domain"`
	PublishedAt    time.Time `json:"published_at"`
	Language       string    `json:"language"`
	SourceCountry  string    `json:"source_country"`
	ImageURL       string    `json:"image_url,omitempty"`
	MatchedSymbols []string  `json:"matched_symbols"`
}

type NewsFeed struct {
	Data   []NewsArticle `json:"data"`
	AsOf   time.Time     `json:"as_of"`
	Source string        `json:"source"`
	Count  int           `json:"count"`
}
