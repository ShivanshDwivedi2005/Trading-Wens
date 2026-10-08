package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
)

const maxResponseSize = 4 << 20

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("invalid NLP service URL")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: parsed.String(), httpClient: httpClient}, nil
}

func (c *Client) Analyze(ctx context.Context, texts []string) ([]domain.SentimentAnalysis, error) {
	if len(texts) == 0 {
		return []domain.SentimentAnalysis{}, nil
	}
	body, err := json.Marshal(inferenceRequest{Texts: texts})
	if err != nil {
		return nil, fmt.Errorf("encode NLP request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/sentiment", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create NLP request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request NLP inference: %w", err)
	}
	defer res.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read NLP response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("NLP inference failed with status %d", res.StatusCode)
	}

	var response inferenceResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, fmt.Errorf("decode NLP response: %w", err)
	}
	if len(response.Predictions) != len(texts) {
		return nil, fmt.Errorf("NLP inference returned %d predictions for %d texts", len(response.Predictions), len(texts))
	}
	results := make([]domain.SentimentAnalysis, len(response.Predictions))
	for index, prediction := range response.Predictions {
		if err := validatePrediction(prediction); err != nil {
			return nil, fmt.Errorf("invalid NLP prediction %d: %w", index, err)
		}
		results[index] = domain.SentimentAnalysis{
			Label:         prediction.Label,
			Score:         prediction.SentimentScore,
			Confidence:    prediction.Confidence,
			Uncertainty:   prediction.Uncertainty,
			Probabilities: prediction.Probabilities,
			Model:         response.Model.Name,
			ModelVersion:  response.Model.Version,
		}
	}
	return results, nil
}

type inferenceRequest struct {
	Texts []string `json:"texts"`
}

type inferenceResponse struct {
	Model struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"model"`
	Predictions []prediction `json:"predictions"`
}

type prediction struct {
	Label          string             `json:"label"`
	Confidence     float64            `json:"confidence"`
	SentimentScore float64            `json:"sentiment_score"`
	Uncertainty    float64            `json:"uncertainty"`
	Probabilities  map[string]float64 `json:"probabilities"`
}

func validatePrediction(value prediction) error {
	if value.Label != "NEGATIVE" && value.Label != "NEUTRAL" && value.Label != "POSITIVE" {
		return errors.New("unsupported sentiment label")
	}
	if value.SentimentScore < -1 || value.SentimentScore > 1 {
		return errors.New("sentiment score must be between -1 and 1")
	}
	if value.Confidence < 0 || value.Confidence > 1 || value.Uncertainty < 0 || value.Uncertainty > 1 {
		return errors.New("confidence and uncertainty must be between 0 and 1")
	}
	if len(value.Probabilities) != 3 {
		return errors.New("three class probabilities are required")
	}
	return nil
}
