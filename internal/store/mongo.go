package store

import (
	"context"
	"time"

	"sentimental/internal/analysis"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// SentimentRecord is a single point-in-time snapshot of a ticker's sentiment.
type SentimentRecord struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Ticker       string        `bson:"ticker"`
	Mentions     int           `bson:"mentions"`
	Score        float64       `bson:"score"`
	Label        string        `bson:"label"`
	ScrapedAt    time.Time     `bson:"scraped_at"`
	NewestPostAt *time.Time    `bson:"newest_post_at,omitempty"`
}

type MongoStore struct {
	col *mongo.Collection
}

func NewMongo(ctx context.Context, uri, collection string) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	col := client.Database("sentimental").Collection(collection)
	return &MongoStore{col: col}, nil
}

func (s *MongoStore) Save(ctx context.Context, results map[string]*analysis.Result) error {
	if len(results) == 0 {
		return nil
	}

	now := time.Now().UTC()
	docs := make([]any, 0, len(results))
	for _, r := range results {
		docs = append(docs, SentimentRecord{
			Ticker:    r.Ticker,
			Mentions:  r.Mentions,
			Score:     r.AverageScore(),
			Label:     r.Label(),
			ScrapedAt: now,
		})
	}

	_, err := s.col.InsertMany(ctx, docs)
	return err
}

func (s *MongoStore) SaveBitcoin(ctx context.Context, result *analysis.Result, newestPostAt time.Time) error {
	now := time.Now().UTC()
	doc := SentimentRecord{
		Ticker:       result.Ticker,
		Mentions:     result.Mentions,
		Score:        result.AverageScore(),
		Label:        result.Label(),
		ScrapedAt:    now,
		NewestPostAt: &newestPostAt,
	}
	_, err := s.col.InsertOne(ctx, doc)
	return err
}
