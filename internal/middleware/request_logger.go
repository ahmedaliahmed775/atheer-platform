package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// RequestLogger is Layer 2 of the transaction pipeline
// Logs structured request metadata without sensitive financial data (NFR-SEC-006)
type RequestLogger struct{}

func NewRequestLogger() *RequestLogger {
	return &RequestLogger{}
}

func (rl *RequestLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		ww := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		slog.Info("Request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.statusCode,
			"duration_ms", duration.Milliseconds(),
			"device_id", r.Header.Get("X-Device-ID"),
			"wallet_id", r.Header.Get("X-Wallet-ID"),
			"channel", r.Header.Get("X-Channel"),
			"ip", r.RemoteAddr,
			"request_id", r.Header.Get("X-Request-ID"),
			// ⚠️ Never log: amount, signatures, seeds, keys
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
