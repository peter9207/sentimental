package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"sentimental/internal/analysis"
	"sentimental/internal/source"
	"sentimental/internal/store"
	"sentimental/internal/ticker"

	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor sentiment for stock tickers across data sources",
	RunE:  runMonitor,
}

func init() {
	monitorCmd.Flags().StringSliceP("subreddits", "s", []string{"wallstreetbets", "stocks", "investing"}, "subreddits to monitor")
	monitorCmd.Flags().IntP("limit", "l", 25, "posts to fetch per subreddit")
	monitorCmd.Flags().Duration("interval", time.Minute, "how often to scrape")
	monitorCmd.Flags().String("mongo", "mongodb://root:password@localhost:27017/sentimental?authSource=admin", "MongoDB connection URI")
}

func runMonitor(cmd *cobra.Command, args []string) error {
	subreddits, _ := cmd.Flags().GetStringSlice("subreddits")
	limit, _ := cmd.Flags().GetInt("limit")
	interval, _ := cmd.Flags().GetDuration("interval")
	mongoURI, _ := cmd.Flags().GetString("mongo")

	if uri := os.Getenv("MONGODB_URI"); uri != "" {
		mongoURI = uri
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	analyzer, err := analysis.New()
	if err != nil {
		return err
	}

	reddit, err := source.NewReddit()
	if err != nil {
		return fmt.Errorf("initializing browser: %w", err)
	}
	defer reddit.Close()

	db, err := store.NewMongo(ctx, mongoURI)
	if err != nil {
		return fmt.Errorf("connecting to mongodb: %w", err)
	}

	fmt.Printf("Monitoring %s every %s. Press Ctrl+C to stop.\n\n", strings.Join(subreddits, ", "), interval)

	// Run immediately, then on each tick
	if err := scrape(ctx, reddit, analyzer, db, subreddits, limit); err != nil {
		fmt.Printf("scrape error: %v\n", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down.")
			return nil
		case <-ticker.C:
			if err := scrape(ctx, reddit, analyzer, db, subreddits, limit); err != nil {
				fmt.Printf("scrape error: %v\n", err)
			}
		}
	}
}

func scrape(ctx context.Context, reddit *source.Reddit, analyzer *analysis.Analyzer, db *store.MongoStore, subreddits []string, limit int) error {
	fmt.Printf("[%s] Starting scrape...\n", time.Now().Format(time.TimeOnly))

	results := make(map[string]*analysis.Result)

	for i, sub := range subreddits {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		fmt.Printf("  Fetching r/%s...\n", sub)
		posts, err := reddit.Fetch(ctx, sub, limit)
		if err != nil {
			fmt.Printf("  warning r/%s: %v\n", sub, err)
			continue
		}

		for _, post := range posts {
			text := post.Title + " " + post.Body
			tickers := ticker.Extract(text)
			if len(tickers) == 0 {
				continue
			}
			score := analyzer.Score(text)
			for _, t := range tickers {
				if _, ok := results[t]; !ok {
					results[t] = &analysis.Result{Ticker: t}
				}
				results[t].Mentions++
				results[t].TotalScore += score
			}
		}
	}

	if err := db.Save(ctx, results); err != nil {
		return fmt.Errorf("saving to mongodb: %w", err)
	}

	printResults(results)
	fmt.Printf("[%s] Saved %d tickers. Next run in ~1m.\n\n", time.Now().Format(time.TimeOnly), len(results))
	return nil
}

func printResults(results map[string]*analysis.Result) {
	if len(results) == 0 {
		fmt.Println("  No ticker mentions found.")
		return
	}

	sorted := make([]*analysis.Result, 0, len(results))
	for _, r := range results {
		sorted = append(sorted, r)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Mentions > sorted[j].Mentions
	})

	fmt.Printf("\n  %-8s  %-8s  %-9s  %-5s\n", "TICKER", "MENTIONS", "SENTIMENT", "SCORE")
	fmt.Println("  " + strings.Repeat("-", 36))
	for _, r := range sorted {
		fmt.Printf("  %-8s  %-8d  %-9s  %.2f\n", "$"+r.Ticker, r.Mentions, r.Label(), r.AverageScore())
	}
	fmt.Println()
}
