// وسيط CORS — السماح بطلبات المتصفح من نطاقات مختلفة
// في التطوير المحلي: localhost:3000 ← localhost:8080

package middleware

import (
	"fmt"
	"net/http"
)

// CORSConfig — إعدادات CORS
type CORSConfig struct {
	AllowedOrigins   []string // النطاقات المسموحة
	AllowedMethods   []string // طرق HTTP المسموحة
	AllowedHeaders   []string // رؤوس HTTP المسموحة
	AllowCredentials bool     // السماح بإرسال الكوكيز
	MaxAge           int      // مدة تخزين الاستجابة المسبقة (بالثواني)
}

// DefaultCORSConfig — إعدادات CORS الافتراضية للتطوير
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID", "X-API-Key"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 ساعة
	}
}

// CORSMiddleware — وسيط CORS يُضيف رؤوس الاستجابة المناسبة
func CORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// التحقق من أن الأصل مسموح
			allowedOrigin := ""
			for _, o := range config.AllowedOrigins {
				if o == origin {
					allowedOrigin = origin
					break
				}
			}

			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-API-Key")
				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
				}
			}

			// معالجة الطلب المسبق (preflight)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
