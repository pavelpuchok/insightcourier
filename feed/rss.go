package feed

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

type RSS struct {
	url    string
	parser *gofeed.Parser
}

func NewRSS(url string) *RSS {
	return &RSS{
		url:    url,
		parser: gofeed.NewParser(),
	}
}

func (rss *RSS) Fetch(ctx context.Context, since time.Time) ([]Item, error) {
	feed, err := rss.parser.ParseURLWithContext(rss.url, ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS feed from %s. %w", rss.url, err)
	}

	result := make([]Item, 0, len(feed.Items))

	for _, it := range feed.Items {
		t := getTime(it)
		if t.Before(since) || t.Equal(since) {
			continue
		}

		result = append(result, Item{
			Source:      rss.url,
			Title:       it.Title,
			Description: it.Description,
			Link:        it.Link,
			Time:        t,
		})
	}

	return result, nil
}

func getTime(it *gofeed.Item) time.Time {
	if it.UpdatedParsed != nil {
		return *it.UpdatedParsed
	}

	if it.PublishedParsed != nil {
		return *it.PublishedParsed
	}

	return time.Now()
}
