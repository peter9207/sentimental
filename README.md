# sentimental

An experiment in using sentiment analysis to analyze stock values. The tool scrapes data sources (currently Reddit) for mentions of stock tickers and scores the surrounding content as bullish, bearish, or neutral. Results are persisted to MongoDB over time to track how sentiment shifts.

## How it works

- Scrapes subreddits (e.g. r/wallstreetbets, r/stocks) using a headless browser
- Detects stock ticker mentions in the format `$TSLA`, `$AAPL`, etc.
- Scores each post using sentiment analysis
- Aggregates scores per ticker and saves snapshots to MongoDB on a recurring interval

## Usage

```bash
docker compose up -d
go run . monitor
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--subreddits` | `wallstreetbets,stocks,investing` | Subreddits to scrape |
| `--limit` | `25` | Posts to fetch per subreddit |
| `--interval` | `1m` | How often to scrape |
| `--mongo` | `mongodb://root:password@localhost:27017/sentimental?authSource=admin` | MongoDB URI |
