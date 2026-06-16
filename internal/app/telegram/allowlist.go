package telegram

import (
	"context"
	"log/slog"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const unauthorizedMessage = "Sorry, Cinna only opens to the internal user for now."

type allowListMiddleware struct {
	allowlist ports.AllowListRepository
	logger    *slog.Logger
}

func AllowListMiddleware(
	allowlist ports.AllowListRepository,
	logger *slog.Logger,
) bot.Middleware {
	m := &allowListMiddleware{
		allowlist: allowlist,
		logger:    logger,
	}
	return m.middleware
}

func (m *allowListMiddleware) middleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, bot *bot.Bot, update *models.Update) {
		logger := m.logger.With("path", "internal/telegram/allowlist/middleware")
		userID, chatID, ok := validateBotUpdate(update)
		if !ok {
			logger.Error("failed to validate bot update.")
			return
		}
		allowed, err := m.allowlist.IsAllowedUser(ctx, userID)
		if err != nil {
			logger.Error("failed to validate user", "error", err)
			return
		}
		if !allowed {
			err := m.sendUnauthorizedMessage(ctx, bot, chatID)
			if err != nil {
				logger.Error("failed to send unauthorized message")
			}
			return
		}
		next(ctx, bot, update)
	}
}

func (m *allowListMiddleware) sendUnauthorizedMessage(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
) error {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   unauthorizedMessage,
	})
	return err
}

// helper functions
func validateBotUpdate(update *models.Update) (userID int64, chatID int64, ok bool) {
	if update == nil || update.Message == nil || update.Message.From == nil {
		return 0, 0, false
	}
	return update.Message.From.ID, update.Message.Chat.ID, true
}
