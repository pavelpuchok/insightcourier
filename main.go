package main

import (
	"context"
	"flag"
	"os"

	"github.com/pavelpuchok/insightcourier/config"
	"github.com/pavelpuchok/insightcourier/feed"
	"github.com/pavelpuchok/insightcourier/planner"
	"github.com/pavelpuchok/insightcourier/storage"
	"github.com/pavelpuchok/insightcourier/tg"
)

func main() {
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

	s, err := storage.NewFileStorage(cfg.FileStorage)
	if err != nil {
		panic(err)
	}

	queue := make(chan Job)

	w := &Worker{
		Queue:    queue,
		Storage:  s,
		Reporter: bot,
	}

	p := &planner.InMemoryPlanner{}

	for _, src := range cfg.RSSSources {
		job := rss(src.FeedURL, queue)
		p.AddJob(context.Background(), src.UpdateInterval, job)
	}

	go w.Process(context.Background())

	select {}
}

func rss(source string, queue chan Job) func() {
	rss := feed.NewRSS(source)
	return func() {
		queue <- Job{
			Source:  source,
			Fetcher: rss,
		}
	}
}
