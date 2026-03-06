package analysis

import (
	"fmt"

	"github.com/cdipaolo/sentiment"
)

// Result holds aggregated sentiment for a single ticker.
type Result struct {
	Ticker     string
	Mentions   int
	TotalScore float64
}

func (r Result) AverageScore() float64 {
	if r.Mentions == 0 {
		return 0
	}
	return r.TotalScore / float64(r.Mentions)
}

func (r Result) Label() string {
	avg := r.AverageScore()
	switch {
	case avg >= 0.6:
		return "Bullish"
	case avg <= 0.4:
		return "Bearish"
	default:
		return "Neutral"
	}
}

// Analyzer scores text using a Naive Bayes model.
type Analyzer struct {
	model sentiment.Models
}

func New() (*Analyzer, error) {
	model, err := sentiment.Restore()
	if err != nil {
		return nil, fmt.Errorf("loading sentiment model: %w", err)
	}
	return &Analyzer{model: model}, nil
}

// Score returns a value in [0,1]: 1 = positive, 0 = negative.
func (a *Analyzer) Score(text string) float64 {
	result := a.model.SentimentAnalysis(text, sentiment.English)
	return float64(result.Score)
}
