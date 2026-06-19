package telegram

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type AgentHandler interface {
	HandleText(
		ctx context.Context,
		userID int64,
		text string,
	) (string, error)
}

type Client struct {
	repositories  *ports.Repositories
	bot           *bot.Bot
	agentHandler  AgentHandler
	webhookURL    string
	webhookSecret string
	logger        *slog.Logger
}

// create a new telegram bot client
func NewClient(
	botToken string,
	repositories *ports.Repositories,
	agentHandler AgentHandler,
	webhookURL string,
	webhookSecret string,
	logger *slog.Logger,
) (*Client, error) {
	if logger == nil {
		logger = slog.Default()
	}
	// create a user lookup set
	client := &Client{
		repositories:  repositories,
		logger:        logger,
		webhookURL:    webhookURL,
		webhookSecret: webhookSecret,
		agentHandler:  agentHandler,
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(client.handleUpdate),
		bot.WithMiddlewares(
			AllowListMiddleware(repositories.AllowList, logger),
		),
		bot.WithWebhookSecretToken(webhookSecret),
	}

	bot, err := bot.New(botToken, opts...)
	if err != nil {
		return nil, err
	}
	client.bot = bot
	return client, nil
}

func (c *Client) SetWebhook(ctx context.Context) error {
	_, err := c.bot.SetWebhook(ctx, &bot.SetWebhookParams{
		URL:         c.webhookURL,
		SecretToken: c.webhookSecret,
	})
	if err != nil {
		return err
	}
	c.logger.Info("telegram webhook configured", "url", c.webhookURL)
	return nil
}

func (c *Client) StartWebhook(ctx context.Context) {
	c.logger.Info("Cinna telegram webhook receiver started")
	c.bot.StartWebhook(ctx)
	// will only log when webhook stopped
	c.logger.Info("Cinna telegram webhook receiver stopped")
}

func (c *Client) RunPolling(ctx context.Context) {
	c.logger.Info("telegram polling started")
	c.bot.Start(ctx)
	c.logger.Info("telegram polling stopped")
}

func (c *Client) WebhookHandler() http.Handler {
	return c.bot.WebhookHandler()
}

// ========== Handlers ==========
func (c *Client) handleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := c.logger.With("path", "internal/telegram/bot/handleUpdate")
	if update.Message == nil || update.Message.Text == "" {
		return
	}
	reply, err := c.agentHandler.HandleText(
		ctx,
		update.Message.From.ID,
		update.Message.Text)
	if err != nil {
		logger.Error(
			"failed to get reply from agent",
			"user_id", update.Message.From.ID,
			"error", err,
		)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "inter server error",
		})
		return
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   reply,
	})
	if err != nil {
		logger.Error(
			"failed to send telegram response",
			"chat_id", update.Message.Chat.ID,
			"error", err,
		)
	}
}
