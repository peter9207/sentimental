package source

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type Reddit struct {
	browser *rod.Browser
}

func NewReddit() (*Reddit, error) {
	url, err := launcher.New().Headless(true).Launch()
	if err != nil {
		return nil, fmt.Errorf("launching browser: %w", err)
	}

	browser := rod.New().ControlURL(url).MustConnect()
	return &Reddit{browser: browser}, nil
}

func (r *Reddit) Name() string { return "reddit" }

func (r *Reddit) Close() error {
	return r.browser.Close()
}

func (r *Reddit) Fetch(ctx context.Context, subreddit string, limit int) ([]Post, error) {
	url := fmt.Sprintf("https://old.reddit.com/r/%s/hot/", subreddit)

	page, err := r.browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, fmt.Errorf("opening page: %w", err)
	}
	defer page.Close()

	page = page.Context(ctx)

	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("waiting for page load: %w", err)
	}

	things, err := page.Elements("div.thing")
	if err != nil {
		return nil, fmt.Errorf("finding posts: %w", err)
	}

	var posts []Post
	for _, thing := range things {
		if len(posts) >= limit {
			break
		}

		id, _ := thing.Attribute("data-fullname")

		titleEl, err := thing.Element("a.title")
		if err != nil {
			continue
		}

		title, err := titleEl.Text()
		if err != nil || strings.TrimSpace(title) == "" {
			continue
		}

		postURL, _ := titleEl.Attribute("href")
		if postURL != nil && strings.HasPrefix(*postURL, "/") {
			full := "https://old.reddit.com" + *postURL
			postURL = &full
		}

		post := Post{
			Title:  strings.TrimSpace(title),
			Source: "reddit/" + subreddit,
		}
		if id != nil {
			post.ID = *id
		}
		if postURL != nil {
			post.URL = *postURL
		}

		posts = append(posts, post)
	}

	return posts, nil
}
