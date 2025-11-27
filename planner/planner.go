package planner

import (
	"context"
	"time"
)

type InMemoryPlanner struct {
}

func (p *InMemoryPlanner) AddJob(ctx context.Context, interval time.Duration, action func()) {
	go func() {
		t := time.NewTicker(interval)

		action()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				action()
			}
		}
	}()
}
