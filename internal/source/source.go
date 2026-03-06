package source

import "context"

// Post represents a piece of content from any data source.
type Post struct {
	ID     string
	Title  string
	Body   string
	Source string
	URL    string
}

// DataSource is implemented by any content provider (Reddit, StockTwits, etc.).
type DataSource interface {
	Name() string
	Fetch(ctx context.Context, subreddit string, limit int) ([]Post, error)
}
