package tg

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/pavelpuchok/insightcourier/config"
	"github.com/pavelpuchok/insightcourier/feed"
)

type Bot struct {
	b      *bot.Bot
	chatId int64
}

func NewBot(cfg config.TelegramConfig) (*Bot, error) {
	b := &Bot{
		chatId: cfg.ChatID,
	}

	bot, err := bot.New(cfg.APIKey, bot.WithCallbackQueryDataHandler("btn;", bot.MatchTypePrefix, b.handleCallback))
	if err != nil {
		return nil, fmt.Errorf("unable to create telegram bot. %w", err)
	}

	b.b = bot

	return b, nil
}

type buttonType int8

const (
	like buttonType = iota
	dislike
)

func (t buttonType) Emoji() string {
	switch t {
	case like:
		return "👍"
	case dislike:
		return "👎"
	default:
		return ""
	}
}

func printButtonData(t buttonType, sourceItemID int32) string {
	return fmt.Sprintf("btn;%d;%d", t, sourceItemID)
}

func parseButtonData(s string) (buttonType, int32, error) {
	if !strings.HasPrefix(s, "btn;") {
		return 0, 0, fmt.Errorf("unexpected button prefix")
	}

	parts := strings.Split(s, ";")
	if len(parts) != 3 {
		return 0, 0, fmt.Errorf("unexpected parts len")
	}

	t, err := strconv.ParseInt(parts[1], 10, 8)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid button type. %w", err)
	}

	sid, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid source item ID. %w", err)
	}

	return buttonType(t), int32(sid), nil
}
func (b *Bot) ListenUpdates(ctx context.Context) {
	b.b.Start(ctx)
}

func (b *Bot) Report(ctx context.Context, it feed.Item, sid int32) error {
	_, err := b.b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: b.chatId,
		Text:   it.Link,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: like.Emoji(), CallbackData: printButtonData(like, sid)},
					{Text: dislike.Emoji(), CallbackData: printButtonData(dislike, sid)},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send TG message. %w", err)
	}
	return nil
}

func (b *Bot) handleCallback(ctx context.Context, _ *bot.Bot, update *models.Update) {
	_, err := b.b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})
	if err != nil {
		slog.Error("failed to answer callback query", slog.String("error", err.Error()))
	}

	err = b.handleButtonCallback(ctx, update)
	if err != nil {
		slog.Error("failed process button callback", slog.String("error", err.Error()))
	}
}

func (b *Bot) handleButtonCallback(ctx context.Context, update *models.Update) error {
	t, sid, err := parseButtonData(update.CallbackQuery.Data)
	if err != nil {
		return fmt.Errorf("failed to parse button data. Data: %s, %w", update.CallbackQuery.Data, err)
	}

	_, err = b.b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    b.chatId,
		MessageID: update.CallbackQuery.Message.Message.ID,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: make([][]models.InlineKeyboardButton, 0),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to hide reply markup. Data: %s, %w", update.CallbackQuery.Data, err)
	}

	b.b.SetMessageReaction(ctx, &bot.SetMessageReactionParams{
		ChatID:    b.chatId,
		MessageID: update.CallbackQuery.Message.Message.ID,
		Reaction: []models.ReactionType{
			{
				Type: models.ReactionTypeTypeEmoji,
				ReactionTypeEmoji: &models.ReactionTypeEmoji{
					Type:  models.ReactionTypeTypeEmoji,
					Emoji: t.Emoji(),
				},
			},
		},
	})

	slog.Info("processed callback query", slog.Int("sourceItemID", int(sid)))

	return nil
}
