package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/app/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// load environment variables
	config, err := app.LoadConfig()
	if err != nil {
		logger.Error("Failed to load env variables", "error", err)
		os.Exit(1)
	}

	// create context and handles exit conditions
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// start telegram bot
	tgClient, err := telegram.NewClient(
		config.TelegramBotToken,
		config.WebhookURL,
		config.WebhookSecret,
		config.AllowedAdminUsers,
		logger,
	)
	if err != nil {
		logger.Error("failed to create telegram client", "error", err)
		os.Exit(1)
	}
	// use long-polling for local development, webhook for production
	if config.RuntimeEnv == app.RuntimeDev {
		tgClient.RunPolling(ctx)
	} else {
		runWebhook(ctx, logger, tgClient)
	}
}

// there's still some issue with this part:
// 1. Enforce https because telegram does not support http
// 2. When press ctrl + C, the terminal(program) doesn't exit
// But this will only be tested when going to the cloud
func runWebhook(
	ctx context.Context,
	logger *slog.Logger,
	tgClient *telegram.Client,
) {
	if err := tgClient.SetWebhook(ctx); err != nil {
		logger.Error("failed to setup telegram webhook", "error", err)
		os.Exit(1)
	}
	go tgClient.StartWebhook(ctx)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Info("Starting http server", "port", port)
	if err := http.ListenAndServe(":"+port, tgClient.WebhookHandler()); err != nil {
		logger.Error("http server failed", "error", err)
		os.Exit(1)
	}
}
