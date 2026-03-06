package ticker

import (
	"regexp"
	"strings"
)

// matches $TSLA, $AAPL, etc.
var tickerRegex = regexp.MustCompile(`\$([A-Z]{1,5})`)

// Extract returns unique ticker symbols found in text.
func Extract(text string) []string {
	matches := tickerRegex.FindAllStringSubmatch(strings.ToUpper(text), -1)
	seen := make(map[string]struct{})
	var tickers []string
	for _, m := range matches {
		t := m[1]
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			tickers = append(tickers, t)
		}
	}
	return tickers
}
