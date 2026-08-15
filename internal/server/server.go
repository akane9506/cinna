package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/akane9506/cinna/internal/app/telegram"
)

type Server struct {
	webhookURL    string
	webhookSecret string
	port          string
	logger        *slog.Logger
	tgClient      *telegram.Client
}

func NewServer(
	tgClient *telegram.Client,
	webhookURL string,
	webhookSecret string,
	logger *slog.Logger) *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return &Server{
		webhookURL:    webhookURL,
		webhookSecret: webhookSecret,
		logger:        logger,
		port:          port,
		tgClient:      tgClient,
	}
}

// for local development, we do long-polling + http
func (s *Server) ServeHTTPDev(ctx context.Context) {
	logger := s.logger.With("path", "interval/app/server/server/ServeHTTPDev")
	mux := http.NewServeMux()
	mux.Handle("/internal/daily-notification", s.handleDailyNotification(ctx))
	server := http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}
	serverErr := make(chan error, 1)
	go s.tgClient.RunPolling(ctx)
	go func() {
		logger.Info("Starting http dev server", "port", s.port)
		serverErr <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		logger.Info("shutting down http server")
		// we pass system ctx to daily notification, because the process does not rely on request's context
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown failed", "error", err)
			os.Exit(1)
		}
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}
}

// for production development, we do tg webhook + http
func (s *Server) ServeHTTPProd(ctx context.Context) {
	logger := s.logger.With("path", "interval/app/server/server/ServeHTTPProd")
	if err := s.tgClient.SetWebhook(ctx); err != nil {
		logger.Error("failed to setup telegram webhook", "error", err)
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.Handle("/", s.tgClient.WebhookHandler())
	// we pass system ctx to daily notification, because the process does not rely on request's context
	mux.Handle("/internal/daily-notification", s.handleDailyNotification(ctx))
	server := http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}
	go s.tgClient.StartWebhook(ctx)
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("Starting http production server", "port", s.port)
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
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}
}
