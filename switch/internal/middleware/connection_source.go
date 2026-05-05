// وسيط تصنيف مصدر الاتصال — يُحدِّد ما إذا الطلب من شبكة الاتصالات أو الإنترنت العام
// يُضاف إلى سياق الطلب ليُستخدم في طبقة التنفيذ لتسجيل مصدر المعاملة
package middleware

import (
	"context"
	"net/http"

	"github.com/atheer/switch/internal/model"
)

// ConnectionSourceMiddleware — وسيط يُعطّي الطلبات بمصدر الاتصال المُحدَّد
// source: "carrier" أو "internet"
func ConnectionSourceMiddleware(source string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// حقن مصدر الاتصال في سياق الطلب
			ctx := context.WithValue(r.Context(), model.ConnectionSourceCtxKey{}, source)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
