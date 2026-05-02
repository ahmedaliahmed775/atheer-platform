// وسيط معرّف الطلب — X-Request-Id
// يضيف معرّف طلب فريد لكل طلب لتتبع المشاكل
package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
)

// سياق معرّف الطلب
type requestIDKey string

const (
	// RequestIDCtxKey — مفتاح السياق لمعرّف الطلب
	RequestIDCtxKey requestIDKey = "request_id"
	// RequestIDHeader — اسم رأس معرّف الطلب
	RequestIDHeader = "X-Request-Id"
)

// RequestIDMiddleware — وسيط يضيف معرّف طلب فريد
// إذا كان الرأس موجوداً يُبقيه، وإلا يُنشئ واحداً جديداً
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}

		// إضافة معرّف الطلب إلى السياق والرأس
		ctx := context.WithValue(r.Context(), RequestIDCtxKey, requestID)
		w.Header().Set(RequestIDHeader, requestID)

		// إضافة معرّف الطلب إلى سجلات slog
		logger := slog.With("request_id", requestID)
		ctx = context.WithValue(ctx, slogKey{}, logger)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// slogKey — مفتاح السياق للمُسجّل
type slogKey struct{}

// GetRequestIDFromContext — يستخرج معرّف الطلب من السياق
func GetRequestIDFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(RequestIDCtxKey).(string); ok {
		return val
	}
	return ""
}

// generateRequestID — يولّد معرّف طلب فريد
func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
