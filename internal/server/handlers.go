package server

import (
	"crypto/subtle"
	"net"
	"net/http"
)

func (s *Server) handleDailyNotification() http.HandlerFunc {
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
		ctx := r.Context()
		if err := s.tgClient.HandleDailyNotification(ctx); err != nil {
			logger.Error("failed to send daily notifications",
				"error", err,
			)
			http.Error(w, "failed to send notification", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
