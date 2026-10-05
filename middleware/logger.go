package middleware

import (
	"log/slog"
	"net/http"
)

func LogRequest(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("http-request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "user", r.UserAgent())
		next.ServeHTTP(w, r)
	})
}
