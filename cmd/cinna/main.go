package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/app/agent"
	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/app/telegram"
	db "github.com/akane9506/cinna/internal/postgres"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/akane9506/cinna/internal/security"
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

	// connect to PostgreSQL
	pool, err := db.Open(ctx, config.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to PostgreSQL server", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
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
	agentMemory := db.NewAgentMemoryRepository(queries, messageCipher, logger)
	repos := &ports.Repositories{
		AllowList:    allowlistRepo,
		ShoppingList: shoppingListRepo,
		AgentMemory:  agentMemory,
	}

	// create an agent
	agent, err := agent.NewCinnaReactAgent(ctx, config, repos, logger)
	if err != nil {
		logger.Error("failed to create cinna agent", "error", err)
		os.Exit(1)
	}

	// start telegram bot
	tgClient, err := telegram.NewClient(
		config.TelegramBotToken,
		repos,
		agent,
		config.WebhookURL,
		config.WebhookSecret,
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
	server := http.Server{
		Addr:    ":" + port,
		Handler: tgClient.WebhookHandler(),
	}
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("Starting http server", "port", port)
		serverErr <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		logger.Info("shutting down http server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown failed", "error", err)
			os.Exit(1)
		}
		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
		logger.Info("http server stopped")
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}
}
