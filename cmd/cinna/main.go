package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/app/agent"
	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/app/telegram"
	db "github.com/akane9506/cinna/internal/postgres"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/akane9506/cinna/internal/security"
	"github.com/akane9506/cinna/internal/server"
	"github.com/akane9506/cinna/internal/utils"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: utils.LoggerReplaceLevelWithSeverity,
	}))
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

	// connect to PostgreSQL
	pool, err := db.Open(ctx, config.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to PostgreSQL server", "error", err)
		os.Exit(1)
	}

	defer func() {
		logger.Info("closing PostgreSQL pool")
		pool.Close()
		logger.Info("PostgreSQL pool closed")
	}()

	// check schemas and tables
	if err := db.CheckSchema(ctx, pool); err != nil {
		logger.Error("database schema check failed", "error", err)
		os.Exit(1)
	}

	// setup message encryption
	messageCipher, err := security.NewMessageCipher(config.MessageEncryptionKeyID, map[string]string{
		config.MessageEncryptionKeyID: config.MessageEncryptionKey,
	})
	if err != nil {
		logger.Error("failed to initialize message cipher", "error", err)
		os.Exit(1)
	}

	// setup repositories
	queries := sqlc.New(pool)
	allowlistRepo := db.NewAllowListRepository(queries)
	shoppingListRepo := db.NewShoppingListRepository(queries)
	agentMemoryRepo := db.NewAgentMemoryRepository(queries, messageCipher, logger)
	feedbackRepo := db.NewFeedbacksRepository(queries)
	repos := &ports.Repositories{
		AllowList:    allowlistRepo,
		ShoppingList: shoppingListRepo,
		AgentMemory:  agentMemoryRepo,
		Feedback:     feedbackRepo,
	}

	// create an agent
	agent, err := agent.NewCinnaReactAgent(ctx, config, repos, logger)
	if err != nil {
		logger.Error("failed to create cinna agent", "error", err)
		os.Exit(1)
	}

	// start telegram bot
	tgClient, err := telegram.NewClient(
		ctx,
		config.TelegramBotToken,
		repos,
		agent,
		config.WebhookURL,
		config.WebhookSecret,
		config.AppTimezone,
		logger,
	)
	if err != nil {
		logger.Error("failed to create telegram client", "error", err)
		os.Exit(1)
	}

	// setup web server
	server := server.NewServer(
		tgClient,
		config.WebhookURL,
		config.WebhookSecret,
		logger,
	)

	// use long-polling for local development, webhook for production
	if config.RuntimeEnv == app.RuntimeDev {
		server.ServeHTTPDev(ctx)
	} else {
		server.ServeHTTPProd(ctx)
	}
}
