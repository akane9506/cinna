package telegram

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/go-telegram/bot"
)

type AgentHandler interface {
	HandleText(
		ctx context.Context,
		userID int64,
		messageTime time.Time,
		text string,
	) (string, error)
	HandleAudio(
		ctx context.Context,
		file *os.File,
	) (string, error)
}

type Client struct {
	repositories  *ports.Repositories
	bot           *bot.Bot
	agentHandler  AgentHandler
	webhookURL    string
	webhookSecret string
	timezone      string
	logger        *slog.Logger
}

// create a new telegram bot client
func NewClient(
	botToken string,
	repositories *ports.Repositories,
	agentHandler AgentHandler,
	webhookURL string,
	webhookSecret string,
	timezone string,
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
		timezone:      timezone,
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
