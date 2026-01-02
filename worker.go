package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"codeberg.org/readeck/go-readability/v2"
	"github.com/pavelpuchok/insightcourier/feed"
	"github.com/pavelpuchok/insightcourier/flaresolverr"
	"github.com/pavelpuchok/insightcourier/storage"
)

type Storage interface {
	BeginTxInContext(ctx context.Context) (context.Context, error)
	CommitTxInContext(ctx context.Context) error
	RollbackTxInContext(ctx context.Context) error
	GetSourceUpdateTime(ctx context.Context, source string) (*time.Time, error)
	SetSourceUpdateTime(ctx context.Context, source string, t time.Time) error
	AddSourceItem(ctx context.Context, item storage.AddSourceItemData) error
}

type Fetcher interface {
	Fetch(context.Context, time.Time) ([]feed.Item, error)
}

type Reporter interface {
	Report(context.Context, feed.Item) error
}

type Job struct {
	SourceName string
}

type Worker struct {
	Queue       <-chan Job
	Storage     Storage
	Reporter    Reporter
	FlareSolver *flaresolverr.FlareSolverr
	Fetchers    map[string]Fetcher
}

func (w *Worker) Process(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-w.Queue:
			ctx, err := w.Storage.BeginTxInContext(ctx)
			if err != nil {
				slog.Error("Failed to begin storage transaction", slog.String("error", err.Error()))
				continue
			}

			err = w.processJob(ctx, job)
			if err != nil {
				if err := w.Storage.RollbackTxInContext(ctx); err != nil {
					slog.Error("Failed to rollback storage transaction", slog.String("error", err.Error()))
				}
				slog.Error("Failed job processing", slog.String("error", err.Error()))
				continue
			}
			err = w.Storage.CommitTxInContext(ctx)
			if err != nil {
				slog.Error("Failed to commit storage transaction", slog.String("error", err.Error()))
				continue
			}
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job Job) error {
	t, err := w.Storage.GetSourceUpdateTime(ctx, job.SourceName)
	if err != nil {
		if !errors.Is(err, storage.ErrSourceNotFound) {
			return fmt.Errorf("fail to read storage. %w", err)
		}
		tt := time.Now().Add(time.Hour * -1)
		t = &tt
	}

	f, has := w.Fetchers[job.SourceName]
	if !has {
		return fmt.Errorf("unable to find Fetcher with name %s", job.SourceName)
	}

	items, err := f.Fetch(ctx, *t)
	if err != nil {
		return fmt.Errorf("fail to fetch feed. %w", err)
	}

	var maxT time.Time = *t
	for _, it := range items {
		if maxT.Before(it.Time) {
			maxT = it.Time
		}

		if err := w.parseContent(ctx, job, it); err != nil {
			return fmt.Errorf("failed to parse content. Link: %s. %w", it.Link, err)
		}

		if err := w.report(ctx, job, it); err != nil {
			return fmt.Errorf("failed to report feed item. Link: %s. %w", it.Link, err)
		}

		w.report(ctx, job, it)
	}

	err = w.Storage.SetSourceUpdateTime(ctx, job.SourceName, maxT)
	if err != nil {
		return fmt.Errorf("fail to update storage. %w", err)
	}

	return nil
}

func (w Worker) report(ctx context.Context, job Job, it feed.Item) error {
	err := w.Reporter.Report(ctx, it)
	if err != nil {
		return fmt.Errorf("fail to report feed item (link: %s): %w", it.Link, err)
	}
	return nil
}

func (w Worker) parseContent(ctx context.Context, job Job, it feed.Item) error {
	fsResp, err := w.FlareSolver.Get(it.Link, flaresolverr.WithDisabledMedia())
	if err != nil {
		return fmt.Errorf("fail to get feed item content: %w", err)
	}

	if fsResp.Status != "ok" {
		return fmt.Errorf("unexpected FlareSolverr status: status=%s message=%s", fsResp.Status, fsResp.Message)
	}

	p := readability.NewParser()
	u, err := url.ParseRequestURI(it.Link)
	if err != nil {
		return fmt.Errorf("failed to parse link")
	}

	article, err := p.Parse(strings.NewReader(fsResp.Solution.Response), u)
	if err != nil {
		return fmt.Errorf("readability failed to parse article: %w", err)
	}

	b := &strings.Builder{}
	err = article.RenderText(b)
	if err != nil {
		return fmt.Errorf("readability failed to render article text: %w", err)
	}

	err = w.Storage.AddSourceItem(ctx, storage.AddSourceItemData{
		SourceName:  job.SourceName,
		URL:         it.Link,
		Title:       article.Title(),
		TextContent: b.String(),
		Excerpt:     article.Excerpt(),
		Language:    article.Language(),
		PublishedAt: it.Time,
	})
	if err != nil {
		return fmt.Errorf("failed to save source item: %w", err)
	}
	return nil
}
