package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/akane9506/cinna/internal/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// load environment variables
	_, err := app.LoadConfig()
	if err != nil {
		logger.Error("Failed to load env variables", "error", err)
		os.Exit(1)
	}

	// create context and handles exit conditions
	_, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
}
