package domain

import "time"

type NewsArticle struct {
	ID             string             `json:"id"`
	Title          string             `json:"title"`
	Summary        string             `json:"summary,omitempty"`
	URL            string             `json:"url"`
	Domain         string             `json:"domain"`
	Provider       string             `json:"provider"`
	PublishedAt    time.Time          `json:"published_at"`
	Language       string             `json:"language"`
	SourceCountry  string             `json:"source_country"`
	ImageURL       string             `json:"image_url,omitempty"`
	MatchedSymbols []string           `json:"matched_symbols"`
	Sentiment      *SentimentAnalysis `json:"sentiment,omitempty"`
}

type NewsFeed struct {
	Data      []NewsArticle `json:"data"`
	AsOf      time.Time     `json:"as_of"`
	Source    string        `json:"source"`
	Providers []string      `json:"providers"`
	Count     int           `json:"count"`
}

type SentimentAnalysis struct {
	Label         string             `json:"label"`
	Score         float64            `json:"score"`
	Confidence    float64            `json:"confidence"`
	Uncertainty   float64            `json:"uncertainty"`
	Probabilities map[string]float64 `json:"probabilities"`
	Model         string             `json:"model"`
	ModelVersion  string             `json:"model_version"`
}
