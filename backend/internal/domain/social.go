package domain

import "time"

type SocialMetrics struct {
	Likes   int `json:"likes"`
	Replies int `json:"replies"`
	Reposts int `json:"reposts"`
	Quotes  int `json:"quotes"`
}

type SocialPost struct {
	ID            string        `json:"id"`
	Text          string        `json:"text"`
	URL           string        `json:"url"`
	AuthorName    string        `json:"author_name"`
	Username      string        `json:"username"`
	AvatarURL     string        `json:"avatar_url,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	Language      string        `json:"language"`
	MatchedSymbol string        `json:"matched_symbol"`
	Metrics       SocialMetrics `json:"metrics"`
}

type SocialFeed struct {
	Data   []SocialPost `json:"data"`
	AsOf   time.Time    `json:"as_of"`
	Source string       `json:"source"`
	Symbol string       `json:"symbol"`
	Count  int          `json:"count"`
}
