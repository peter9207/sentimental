package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"sentimental/internal/analysis"
	"sentimental/internal/source"
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
}

func runMonitor(cmd *cobra.Command, args []string) error {
	subreddits, _ := cmd.Flags().GetStringSlice("subreddits")
	limit, _ := cmd.Flags().GetInt("limit")

	analyzer, err := analysis.New()
	if err != nil {
		return err
	}

	reddit, err := source.NewReddit()
	if err != nil {
		return fmt.Errorf("initializing browser: %w", err)
	}
	defer reddit.Close()

	ctx := context.Background()

	results := make(map[string]*analysis.Result)

	for i, sub := range subreddits {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		fmt.Printf("Fetching r/%s...\n", sub)
		posts, err := reddit.Fetch(ctx, sub, limit)
		if err != nil {
			fmt.Printf("  warning: %v\n", err)
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

	printResults(results)
	return nil
}

func printResults(results map[string]*analysis.Result) {
	if len(results) == 0 {
		fmt.Println("No ticker mentions found.")
		return
	}

	// Sort by mentions descending
	sorted := make([]*analysis.Result, 0, len(results))
	for _, r := range results {
		sorted = append(sorted, r)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Mentions > sorted[j].Mentions
	})

	fmt.Println()
	fmt.Printf("%-8s  %-8s  %-9s  %-5s\n", "TICKER", "MENTIONS", "SENTIMENT", "SCORE")
	fmt.Println(strings.Repeat("-", 38))
	for _, r := range sorted {
		fmt.Printf("%-8s  %-8d  %-9s  %.2f\n", "$"+r.Ticker, r.Mentions, r.Label(), r.AverageScore())
	}
}
