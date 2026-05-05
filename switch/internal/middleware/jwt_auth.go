// وسيط المصادقة JWT — التحقق من الرمز والدور والنطاق
// يُرجى الرجوع إلى SPEC §5 — Admin APIs
// WALLET_ADMIN يرى بيانات محفظته فقط (scope filtering)
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/atheer/switch/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

// مفاتيح السياق للمستخدم الإداري
type adminCtxKey string

const (
	// AdminEmailCtxKey — مفتاح السياق لبريد المستخدم الإداري
	AdminEmailCtxKey adminCtxKey = "admin_email"
	// AdminRoleCtxKey — مفتاح السياق لدور المستخدم الإداري
	AdminRoleCtxKey adminCtxKey = "admin_role"
	// AdminScopeCtxKey — مفتاح السياق لنطاق المستخدم الإداري
	AdminScopeCtxKey adminCtxKey = "admin_scope"
)

// AdminClaims — بيانات الرمز المميز JWT للمستخدم الإداري
type AdminClaims struct {
	Email string `json:"email"` // البريد الإلكتروني
	Role  string `json:"role"`  // الدور: SUPER_ADMIN, ADMIN, WALLET_ADMIN, VIEWER
	Scope string `json:"scope"` // النطاق: * أو محفظة محددة مثل jawali
	jwt.RegisteredClaims
}

// JWTAuthMiddleware — وسيط يتحقق من JWT ويدرج بيانات المستخدم في السياق
// يتحقق من: وجود الرمز، صلاحيته، عدم انتهائه
// يُدرج في السياق: البريد، الدور، النطاق
func JWTAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	secret := []byte(jwtSecret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// مسارات المصادقة لا تحتاج JWT
			if strings.HasPrefix(r.URL.Path, "/admin/v1/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			// استخراج الرمز من رأس Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				slog.Warn("وسيط JWT: رأس Authorization مفقود")
				writeAdminError(w, model.NewAppError(model.ErrUnauthorized))
				return
			}

			// التحقق من صيغة Bearer <token>
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				slog.Warn("وسيط JWT: صيغة الرأس غير صحيحة")
				writeAdminError(w, model.NewAppError(model.ErrUnauthorized))
				return
			}

			tokenStr := parts[1]

			// تحليل والتحقق من الرمز
			claims := &AdminClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				// التحقق من خوارزمية التوقيع
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					slog.Warn("وسيط JWT: خوارزمية توقيع غير متوقعة", "alg", token.Header["alg"])
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			})

			if err != nil {
				slog.Warn("وسيط JWT: رمز غير صالح", "error", err)
				if strings.Contains(err.Error(), "token is expired") {
					writeAdminError(w, model.NewAppError(model.ErrTokenExpired))
				} else {
					writeAdminError(w, model.NewAppError(model.ErrUnauthorized))
				}
				return
			}

			if !token.Valid {
				slog.Warn("وسيط JWT: رمز غير صالح")
				writeAdminError(w, model.NewAppError(model.ErrUnauthorized))
				return
			}

			// إدراج بيانات المستخدم في السياق
			ctx := r.Context()
			ctx = context.WithValue(ctx, AdminEmailCtxKey, claims.Email)
			ctx = context.WithValue(ctx, AdminRoleCtxKey, claims.Role)
			ctx = context.WithValue(ctx, AdminScopeCtxKey, claims.Scope)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole — وسيط يتحقق من أن المستخدم يملك الدور المطلوب على الأقل
func RequireRole(minRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetAdminRoleFromContext(r.Context())
			if role == "" {
				writeAdminError(w, model.NewAppError(model.ErrUnauthorized))
				return
			}

			if !model.CanAccess(role, minRole) {
				slog.Warn("وسيط JWT: الدور لا يملك صلاحية", "role", role, "required", minRole)
				writeAdminError(w, model.NewAppError(model.ErrForbiddenRole))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ScopeFilter — يُرجع معرّف المحفظة من النطاق إذا كان المستخدم WALLET_ADMIN
// إذا كان النطاق "*" يُرجع سلسلة فارغة (يعني كل المحافظ)
func ScopeFilter(ctx context.Context) string {
	role := GetAdminRoleFromContext(ctx)
	scope := GetAdminScopeFromContext(ctx)

	// SUPER_ADMIN و ADMIN يرون كل شيء
	if role == model.RoleSuperAdmin || role == model.RoleAdmin {
		return ""
	}

	// WALLET_ADMIN يرى محفظته فقط
	if role == model.RoleWalletAdmin && scope != "*" {
		return scope
	}

	// VIEWER يرى كل شيء (قراءة فقط)
	return ""
}

// GetAdminEmailFromContext — يستخرج بريد المستخدم الإداري من السياق
func GetAdminEmailFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(AdminEmailCtxKey).(string); ok {
		return val
	}
	return ""
}

// GetAdminRoleFromContext — يستخرج دور المستخدم الإداري من السياق
func GetAdminRoleFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(AdminRoleCtxKey).(string); ok {
		return val
	}
	return ""
}

// GetAdminScopeFromContext — يستخرج نطاق المستخدم الإداري من السياق
func GetAdminScopeFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(AdminScopeCtxKey).(string); ok {
		return val
	}
	return ""
}

// writeAdminError — يكتب استجابة خطأ بصيغة JSON
func writeAdminError(w http.ResponseWriter, appErr *model.AppError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(appErr.HTTPStatus)
	// كتابة يدوية لتجنب استيراد حزمة api
	w.Write([]byte(`{"errorCode":"` + appErr.Code + `","errorMessage":"` + appErr.Message + `"}`))
}
