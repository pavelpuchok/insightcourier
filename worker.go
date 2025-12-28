package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/pavelpuchok/insightcourier/feed"
	"github.com/pavelpuchok/insightcourier/storage"
)

type Storage interface {
	GetSourceUpdateTime(ctx context.Context, source string) (*time.Time, error)
	SetSourceUpdateTime(ctx context.Context, source string, t time.Time) error
}

type Fetcher interface {
	Fetch(context.Context, time.Time) ([]feed.Item, error)
}

type Reporter interface {
	Report(context.Context, feed.Item) error
}

type Job struct {
	SourceName string
	Fetcher    Fetcher
}

type Worker struct {
	Queue    <-chan Job
	Storage  Storage
	Reporter Reporter
}

func (w *Worker) Process(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-w.Queue:
			t, err := w.Storage.GetSourceUpdateTime(ctx, job.SourceName)
			if err != nil {
				if !errors.Is(err, storage.ErrSourceNotFound) {
					slog.Error("Failed to read storage", slog.String("error", err.Error()))
					continue
				}
				tt := time.Now().Add(time.Hour * -1)
				t = &tt
			}

			items, err := job.Fetcher.Fetch(ctx, *t)
			if err != nil {
				slog.Error("Failed to fetch feed", slog.String("error", err.Error()))
				continue
			}

			var maxT time.Time = *t
			for _, it := range items {
				if maxT.Before(it.Time) {
					maxT = it.Time
				}

				err := w.Reporter.Report(ctx, it)
				if err != nil {
					slog.Error("Failed to report feed item", slog.String("error", err.Error()))
					continue
				}
			}

			err = w.Storage.SetSourceUpdateTime(ctx, job.SourceName, maxT)
			if err != nil {
				slog.Error("Failed to update storage", slog.String("error", err.Error()))
				continue
			}
		}
	}
}
