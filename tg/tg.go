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
	"github.com/pavelpuchok/insightcourier/storage/psql"
)

type Storage interface {
	BeginTxInContext(ctx context.Context) (context.Context, error)
	CommitTxInContext(ctx context.Context) error
	RollbackTxInContext(ctx context.Context) error
	CreateReaction(ctx context.Context, sourceItemID int32, reactionType psql.ReactionsType) error
}

type Bot struct {
	b       *bot.Bot
	storage Storage
	chatId  int64
}

func NewBot(storage Storage, cfg config.TelegramConfig) (*Bot, error) {
	b := &Bot{
		chatId:  cfg.ChatID,
		storage: storage,
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
	buttonTypeLike buttonType = iota
	buttonTypeDislike
)

func (t buttonType) Emoji() string {
	switch t {
	case buttonTypeLike:
		return "👍"
	case buttonTypeDislike:
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
					{Text: buttonTypeLike.Emoji(), CallbackData: printButtonData(buttonTypeLike, sid)},
					{Text: buttonTypeDislike.Emoji(), CallbackData: printButtonData(buttonTypeDislike, sid)},
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

	cctx, err := b.storage.BeginTxInContext(ctx)
	if err != nil {
		slog.Error("failed to initiate transaction for callback query processing", slog.String("error", err.Error()))
		return
	}

	err = b.handleButtonCallback(cctx, update)
	if err != nil {
		slog.Error("failed process button callback", slog.String("error", err.Error()))
		if err := b.storage.RollbackTxInContext(cctx); err != nil {
			slog.Error("failed to rollback transaction", slog.String("error", err.Error()))
		}
		return
	}

	err = b.storage.CommitTxInContext(cctx)
	if err != nil {
		slog.Error("failed to commit transaction for query callback", slog.String("error", err.Error()))
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

	var rt psql.ReactionsType
	switch t {
	case buttonTypeLike:
		rt = psql.ReactionsTypeLike
	case buttonTypeDislike:
		rt = psql.ReactionsTypeDislike
	}

	err = b.storage.CreateReaction(ctx, sid, rt)
	if err != nil {
		return fmt.Errorf("failed to create reaction in storage. %w", err)
	}

	return nil
}
