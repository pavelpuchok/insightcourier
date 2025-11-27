package feed

import "time"

type Item struct {
	Source      string
	Title       string
	Description string
	Link        string
	Time        time.Time
}
