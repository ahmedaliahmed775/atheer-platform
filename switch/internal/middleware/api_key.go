// وسيط التحقق من مفتاح API — X-API-Key
// يبحث عن المفتاح في جدول wallet_configs
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/model"
)

// سياق المفتاح — نوع مفتاح السياق لتجنب التصادم
type contextKey string

const (
	// APIKeyCtxKey — مفتاح السياق لمفتاح API
	APIKeyCtxKey contextKey = "api_key"
	// WalletIDCtxKey — مفتاح السياق لمعرّف المحفظة
	WalletIDCtxKey contextKey = "wallet_id"
)

// APIKeyMiddleware — وسيط يتحقق من وجود X-API-Key ويبحث عنه في wallet_configs
func APIKeyMiddleware(walletRepo db.WalletRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				slog.Warn("وسيط API: مفتاح API مفقود")
				writeUnauthorized(w)
				return
			}

			// البحث عن المحفظة بمفتاح API
			// ملاحظة: هذا تنفيذ مبسّط — في الإنتاج يُفضل استخدام فهرس عكسي أو ذاكرة تخزين مؤقت
			wallets, err := walletRepo.List(r.Context())
			if err != nil {
				slog.Error("وسيط API: فشل البحث عن المحافظ", "error", err)
				http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
				return
			}

			var matchedWalletId string
			for _, cfg := range wallets {
				if cfg.APIKey == apiKey && cfg.IsActive {
					matchedWalletId = cfg.WalletId
					break
				}
			}

			if matchedWalletId == "" {
				slog.Warn("وسيط API: مفتاح API غير صالح")
				writeUnauthorized(w)
				return
			}

			// إضافة معلومات السياق
			ctx := context.WithValue(r.Context(), APIKeyCtxKey, apiKey)
			ctx = context.WithValue(ctx, WalletIDCtxKey, matchedWalletId)

			slog.Debug("وسيط API: مصادقة ناجحة", "walletId", matchedWalletId)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeUnauthorized — يكتب استجابة 401
func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(model.NewAppError(model.ErrUnauthorized).HTTPStatus)
	w.Write([]byte(`{"errorCode":"UNAUTHORIZED","errorMessage":"غير مُصرَّح — مفتاح API مفقود أو غير صالح"}`))
}

// GetAPIKeyFromContext — يستخرج مفتاح API من السياق
func GetAPIKeyFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(APIKeyCtxKey).(string); ok {
		return val
	}
	return ""
}

// GetWalletIDFromContext — يستخرج معرّف المحفظة من السياق
func GetWalletIDFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(WalletIDCtxKey).(string); ok {
		return val
	}
	return ""
}
