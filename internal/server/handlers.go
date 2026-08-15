package server

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"time"
)

func (s *Server) handleDailyNotification(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := s.logger.With("path", "internal/app/server/handler/handleNotification")
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			logger.Error("Failed to get ip from the request", "error", err)
			ip = r.RemoteAddr
		}
		if r.Method != http.MethodPost {
			logger.Error("method not allowed", "method", r.Method, "ip", ip)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		requestSecret := r.Header.Get("X-Cinna-Webhook-Secret")
		if subtle.ConstantTimeCompare(
			[]byte(requestSecret),
			[]byte(s.webhookSecret),
		) != 1 {
			logger.Error("unauthorized request", "request_secret", requestSecret)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		go func() {
			localCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()
			if err := s.tgClient.HandleDailyNotification(localCtx); err != nil {
				logger.Error("failed to send daily notifications",
					"error", err,
				)
			}
		}()
	}
}
