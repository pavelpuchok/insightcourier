package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/pavelpuchok/insightcourier/config"
	"github.com/pavelpuchok/insightcourier/feed"
	"github.com/pavelpuchok/insightcourier/flaresolverr"
	"github.com/pavelpuchok/insightcourier/planner"
	"github.com/pavelpuchok/insightcourier/storage"
	"github.com/pavelpuchok/insightcourier/tg"
)

func main() {
	ctx := context.Background()

	cfgPath := flag.String("config", os.Getenv("IC_CONFIG_PATH"), "path to config file")
	flag.Parse()

	if *cfgPath == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	cfg, err := config.Load(*cfgPath, config.EnvVarProvider{LookupEnv: os.LookupEnv})
	if err != nil {
		panic(err)
	}

	bot, err := tg.NewBot(cfg.Telegram)
	if err != nil {
		panic(err)
	}

	queue := make(chan Job)

	s, err := storage.NewPostgreSQL(ctx, cfg.PSQLStorage)
	if err != nil {
		panic(err)
	}

	p := &planner.InMemoryPlanner{}

	for name := range cfg.RSSSources {
		id, err := s.CreateSource(ctx, name)
		if err != nil {
			if errors.Is(err, storage.ErrSourceAlreadyExists) {
				slog.Debug("Source already created", slog.String("source.name", name))
				continue
			}
			panic(err)
		}
		slog.Info("New source created", slog.String("source.name", name), slog.Int("source.id", int(id)))
	}

	w := &Worker{
		Queue:       queue,
		Storage:     s,
		Reporter:    bot,
		Fetchers:    make(map[string]Fetcher, len(cfg.RSSSources)),
		FlareSolver: &flaresolverr.FlareSolverr{URL: cfg.FlareSolverr.URL},
	}

	for name, src := range cfg.RSSSources {
		w.Fetchers[name] = feed.NewRSS(src.FeedURL)
		enqeueJob := func() {
			queue <- Job{
				SourceName: name,
			}
		}

		p.AddJob(context.Background(), src.UpdateInterval, enqeueJob)
	}

	go w.Process(context.Background())

	select {}
}
