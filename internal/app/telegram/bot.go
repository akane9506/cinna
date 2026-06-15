package telegram

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Client struct {
	allowedUsers  map[int64]struct{}
	webhookURL    string
	webhookSecret string
	logger        *slog.Logger
	bot           *bot.Bot
	handler       Handler
}

// create a new telegram bot client
func NewClient(
	botToken string,
	webhookURL string,
	webhookSecret string,
	allowedUsers []int64,
	// handler Handler,
	logger *slog.Logger,
) (*Client, error) {
	if logger == nil {
		logger = slog.Default()
	}
	// create a user lookup set
	allowed := make(map[int64]struct{})
	for _, userID := range allowedUsers {
		allowed[userID] = struct{}{}
	}
	client := &Client{
		allowedUsers:  allowed,
		logger:        logger,
		webhookURL:    webhookURL,
		webhookSecret: webhookSecret,
		// handler:      ,
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
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

// ========== helper functions ==========

func (c *Client) isAllowed(userID int64) bool {
	// if no allowed users, basically allow open access
	if len(c.allowedUsers) == 0 {
		return true
	}
	_, ok := c.allowedUsers[userID]
	return ok
}

// temp handler
func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}

// next step, setup a db and cache for adding/updating/removing allowed users
