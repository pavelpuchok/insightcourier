package tg

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelpuchok/insightcourier/config"
	"github.com/pavelpuchok/insightcourier/feed"
)

type Bot struct {
	b      *tgbotapi.BotAPI
	chatId int64
}

func NewBot(cfg config.TelegramConfig) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("unable to create telegram bot. %w", err)
	}

	return &Bot{
		b:      bot,
		chatId: cfg.ChatID,
	}, nil
}

func (b *Bot) Report(ctx context.Context, it feed.Item) error {
	_, err := b.b.Send(tgbotapi.NewMessage(b.chatId, tgbotapi.EscapeText(tgbotapi.ModeHTML, fmt.Sprintf("%s\n%s", it.Title, it.Link))))
	if err != nil {
		return fmt.Errorf("unable to send telegram message. %w", err)
	}
	return nil
}
