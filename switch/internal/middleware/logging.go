// وسيط تسجيل الطلبات — يستخدم slog بدون بيانات حساسة
package middleware

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// responseWriter — غلاف لتسجيل حالة الاستجابة
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader — يلتقط حالة الاستجابة
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack — يدعم http.Hijacker لاتصالات WebSocket
// يُفوّض الاستدعاء للكاتب الأصلي إذا كان يدعم Hijacker
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// LoggingMiddleware — وسيط يسجّل كل طلب HTTP
// يسجّل: الطريقة، المسار، حالة الاستجابة، المدة
// لا يسجّل: جسم الطلب، الرؤوس الحساسة، مفاتيح API
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// غلاف ResponseWriter لالتقاط حالة الاستجابة
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // افتراضي
		}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		// تسجيل الطلب — بدون بيانات حساسة
		slog.Info("طلب HTTP",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}
