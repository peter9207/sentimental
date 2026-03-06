package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sentimental/internal/analysis"
	"sentimental/internal/source"
	"sentimental/internal/store"

	"github.com/spf13/cobra"
)

var bitcoinCmd = &cobra.Command{
	Use:   "bitcoin",
	Short: "Monitor overall sentiment across bitcoin and crypto subreddits",
	RunE:  runBitcoin,
}

func init() {
	bitcoinCmd.Flags().IntP("limit", "l", 25, "posts to fetch per subreddit")
	bitcoinCmd.Flags().Duration("interval", time.Minute, "how often to scrape")
	bitcoinCmd.Flags().String("mongo", "mongodb://root:password@localhost:27017/sentimental?authSource=admin", "MongoDB connection URI")
}

var bitcoinSubreddits = []string{"bitcoin", "CryptoCurrency"}

func runBitcoin(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	interval, _ := cmd.Flags().GetDuration("interval")
	mongoURI, _ := cmd.Flags().GetString("mongo")

	if uri := os.Getenv("MONGODB_URI"); uri != "" {
		mongoURI = uri
	}

	fmt.Printf("[%s] Starting bitcoin monitor\n", time.Now().Format(time.TimeOnly))
	fmt.Printf("  Subreddits : %s\n", "bitcoin, CryptoCurrency")
	fmt.Printf("  Posts/sub  : %d\n", limit)
	fmt.Printf("  Interval   : %s\n\n", interval)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("[%s] Initializing sentiment analyzer...\n", time.Now().Format(time.TimeOnly))
	analyzer, err := analysis.New()
	if err != nil {
		return err
	}

	fmt.Printf("[%s] Launching Reddit source...\n", time.Now().Format(time.TimeOnly))
	reddit, err := source.NewReddit()
	if err != nil {
		return fmt.Errorf("initializing browser: %w", err)
	}
	defer reddit.Close()

	fmt.Printf("[%s] Connecting to MongoDB...\n", time.Now().Format(time.TimeOnly))
	db, err := store.NewMongo(ctx, mongoURI, "bitcoin_sentiment")
	if err != nil {
		return fmt.Errorf("connecting to mongodb: %w", err)
	}

	fmt.Printf("[%s] Ready. Press Ctrl+C to stop.\n\n", time.Now().Format(time.TimeOnly))

	if err := scrapeBitcoin(ctx, reddit, analyzer, db, limit); err != nil {
		fmt.Printf("scrape error: %v\n", err)
	}

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down.")
			return nil
		case <-t.C:
			if err := scrapeBitcoin(ctx, reddit, analyzer, db, limit); err != nil {
				fmt.Printf("scrape error: %v\n", err)
			}
		}
	}
}

func scrapeBitcoin(ctx context.Context, reddit *source.Reddit, analyzer *analysis.Analyzer, db *store.MongoStore, limit int) error {
	fmt.Printf("[%s] Starting scrape...\n", time.Now().Format(time.TimeOnly))

	var total float64
	var count int
	var newestPostAt time.Time

	for i, sub := range bitcoinSubreddits {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		fmt.Printf("  Fetching r/%s...\n", sub)
		posts, err := reddit.Fetch(ctx, sub, limit)
		if err != nil {
			fmt.Printf("  warning r/%s: %v\n", sub, err)
			continue
		}
		fmt.Printf("  r/%s: fetched %d posts\n", sub, len(posts))

		for _, post := range posts {
			text := post.Title + " " + post.Body
			total += analyzer.Score(text)
			count++
			if post.CreatedAt.After(newestPostAt) {
				newestPostAt = post.CreatedAt
			}
		}
	}

	if count == 0 {
		fmt.Println("  No posts found.")
		return nil
	}

	avg := total / float64(count)
	result := &analysis.Result{
		Ticker:     "BTC",
		Mentions:   count,
		TotalScore: total,
	}

	if err := db.SaveBitcoin(ctx, result, newestPostAt); err != nil {
		return fmt.Errorf("saving to mongodb: %w", err)
	}

	label := result.Label()
	fmt.Printf("\n  Posts scored  : %d\n", count)
	fmt.Printf("  Avg score     : %.4f\n", avg)
	fmt.Printf("  Sentiment     : %s\n", label)
	if !newestPostAt.IsZero() {
		fmt.Printf("  Newest post   : %s\n", newestPostAt.Format(time.DateTime))
	}
	fmt.Println()

	return nil
}
