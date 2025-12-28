package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type EnvVarProvider struct {
	LookupEnv func(string) (string, bool)
}

type RSSSourceConfig struct {
	FeedURL        string        `json:"feedUrl"`
	UpdateInterval time.Duration `json:"updateInterval"`
}

type PSQLStorageConfig struct {
	ConnString     string        `json:"-"`
	DefaultTimeout time.Duration `json:"defaulTimeout"`
}

type TelegramConfig struct {
	APIKey string `json:"-"`
	ChatID int64  `json:"chatId"`
}

type Config struct {
	RSSSources  map[string]RSSSourceConfig `json:"rssSources"`
	Telegram    TelegramConfig             `json:"telegram"`
	PSQLStorage PSQLStorageConfig          `json:"psqlStorage"`
}

var (
	DefaultRSSUpdateInterval = 5 * time.Minute
	DefaultPSQLTimeout       = 5 * time.Second
)

func Load(path string, env EnvVarProvider) (*Config, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file. %w", err)
	}

	d := json.NewDecoder(f)
	var cfg Config
	err = d.Decode(&cfg)

	if err != nil {
		return nil, fmt.Errorf("unable decode config. %w", err)
	}

	apiKey, has := env.LookupEnv("IC_TG_BOT_API_KEY")
	if !has || apiKey == "" {
		return nil, errors.New("environment variable IC_TG_BOT_API_KEY should be set to non empty value")
	}
	cfg.Telegram.APIKey = apiKey

	cfg.PSQLStorage.ConnString, _ = env.LookupEnv("IC_PSQL_CONNECTION_STRING")
	if cfg.PSQLStorage.DefaultTimeout == 0 {
		cfg.PSQLStorage.DefaultTimeout = DefaultPSQLTimeout
	}

	for i := range cfg.RSSSources {
		if cfg.RSSSources[i].UpdateInterval == 0 {
			c := cfg.RSSSources[i]
			if c.UpdateInterval == 0 {
				c.UpdateInterval = DefaultRSSUpdateInterval
				cfg.RSSSources[i] = c
			}
		}
	}

	return &cfg, nil
}
