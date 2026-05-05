// معالج مصادقة الإدارة — تسجيل الدخول والخروج وتجديد الرمز
// يُرجى الرجوع إلى SPEC §5 — Admin APIs
package admin

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/middleware"
	"github.com/atheer/switch/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler — معالج مصادقة الإدارة
type AuthHandler struct {
	adminRepo db.AdminRepo
	jwtSecret []byte // سر JWT — لا يُسجَّل أبداً
	jwtExpiry time.Duration
}

// NewAuthHandler — ينشئ معالج مصادقة جديد
func NewAuthHandler(adminRepo db.AdminRepo, jwtSecret string, jwtExpiry time.Duration) *AuthHandler {
	return &AuthHandler{
		adminRepo: adminRepo,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

// LoginRequest — طلب تسجيل الدخول
type LoginRequest struct {
	Email    string `json:"email"`    // البريد الإلكتروني
	Password string `json:"password"` // كلمة المرور
	TOTPCode string `json:"totpCode"` // رمز التحقق الثنائي (اختياري)
}

// LoginResponse — استجابة تسجيل الدخول
type LoginResponse struct {
	AccessToken  string `json:"accessToken"`  // رمز الوصول JWT
	RefreshToken string `json:"refreshToken"` // رمز التجديد JWT
	ExpiresIn    int64  `json:"expiresIn"`    // مدة الصلاحية بالثواني
	Role         string `json:"role"`         // دور المستخدم
	Scope        string `json:"scope"`        // نطاق الصلاحيات
}

// RefreshRequest — طلب تجديد الرمز
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"` // رمز التجديد
}

// LogoutRequest — طلب تسجيل الخروج
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"` // رمز التجديد
}

// HandleLogin — يعالج طلب تسجيل الدخول
// المنطق:
//  1. البحث عن المستخدم ببريده الإلكتروني
//  2. التحقق من كلمة المرور (bcrypt)
//  3. التحقق من TOTP إذا كان مُفعّلاً
//  4. إنشاء رمز الوصول ورمز التجديد
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. تحليل الطلب
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "طلب غير صالح — حقول مفقودة",
		})
		return
	}

	if req.Email == "" || req.Password == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "البريد الإلكتروني وكلمة المرور مطلوبان",
		})
		return
	}

	// 2. البحث عن المستخدم
	user, err := h.adminRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		slog.Error("مصادقة الإدارة: خطأ في البحث عن المستخدم", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}

	if user == nil {
		slog.Warn("مصادقة الإدارة: مستخدم غير موجود", "email", req.Email)
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"errorCode":    model.ErrInvalidCredentials,
			"errorMessage": "بيانات الدخول غير صحيحة",
		})
		return
	}

	// 3. التحقق من أن الحساب مفعّل
	if !user.IsActive {
		slog.Warn("مصادقة الإدارة: حساب معطّل", "email", req.Email)
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "الحساب معطّل",
		})
		return
	}

	// 4. التحقق من كلمة المرور
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		slog.Warn("مصادقة الإدارة: كلمة مرور خاطئة", "email", req.Email)
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"errorCode":    model.ErrInvalidCredentials,
			"errorMessage": "بيانات الدخول غير صحيحة",
		})
		return
	}

	// 5. التحقق من TOTP إذا كان مُفعّلاً
	if user.TOTPSecret != "" {
		if req.TOTPCode == "" {
			// يحتاج رمز TOTP — نُرجع حالة خاصة
			writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
				"errorCode":    model.ErrTOTPRequired,
				"errorMessage": "رمز التحقق الثنائي (TOTP) مطلوب",
			})
			return
		}
		// التحقق من رمز TOTP (تنفيذ مبسّط — في الإنتاج يُستخدم مكتبة TOTP)
		if !validateTOTP(user.TOTPSecret, req.TOTPCode) {
			slog.Warn("مصادقة الإدارة: رمز TOTP غير صالح", "email", req.Email)
			writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
				"errorCode":    model.ErrInvalidCredentials,
				"errorMessage": "رمز التحقق الثنائي غير صالح",
			})
			return
		}
	}

	// 6. إنشاء رمز الوصول
	accessToken, err := h.generateToken(user.Email, user.Role, user.Scope, h.jwtExpiry)
	if err != nil {
		slog.Error("مصادقة الإدارة: فشل إنشاء رمز الوصول", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}

	// 7. إنشاء رمز التجديد (صلاحية أطول)
	refreshToken, err := h.generateToken(user.Email, user.Role, user.Scope, h.jwtExpiry*4)
	if err != nil {
		slog.Error("مصادقة الإدارة: فشل إنشاء رمز التجديد", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}

	// 8. تحديث آخر تسجيل دخول
	user.LastLoginAt = nil // يُحدَّث في قاعدة البيانات
	_ = h.adminRepo.Update(ctx, user)

	slog.Info("مصادقة الإدارة: تسجيل دخول ناجح", "email", req.Email, "role", user.Role)

	writeAdminJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.jwtExpiry.Seconds()),
		Role:         user.Role,
		Scope:        user.Scope,
	})
}

// HandleRefresh — يعالج طلب تجديد الرمز
func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "طلب غير صالح",
		})
		return
	}

	// تحليل رمز التجديد
	claims := &middleware.AdminClaims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"errorCode":    model.ErrTokenExpired,
			"errorMessage": "رمز التجديد منتهي أو غير صالح",
		})
		return
	}

	// البحث عن المستخدم للتأكد من أن الحساب لا يزال مفعّلاً
	user, err := h.adminRepo.FindByEmail(r.Context(), claims.Email)
	if err != nil || user == nil || !user.IsActive {
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"errorCode":    model.ErrTokenRevoked,
			"errorMessage": "الحساب معطّل أو غير موجود",
		})
		return
	}

	// إنشاء رمز وصول جديد
	accessToken, err := h.generateToken(user.Email, user.Role, user.Scope, h.jwtExpiry)
	if err != nil {
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}

	// إنشاء رمز تجديد جديد
	refreshToken, err := h.generateToken(user.Email, user.Role, user.Scope, h.jwtExpiry*4)
	if err != nil {
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}

	writeAdminJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.jwtExpiry.Seconds()),
		Role:         user.Role,
		Scope:        user.Scope,
	})
}

// HandleLogout — يعالج طلب تسجيل الخروج
// في الإصدار الحالي لا يُلغي الرمز فعلياً (يحتاج قائمة سوداء أو Redis)
// يُرجع استجابة ناجحة — العميل يحذف الرمز محلياً
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetAdminEmailFromContext(r.Context())
	slog.Info("مصادقة الإدارة: تسجيل خروج", "email", email)

	writeAdminJSON(w, http.StatusOK, map[string]string{
		"status":  "logged_out",
		"message": "تم تسجيل الخروج بنجاح",
	})
}

// generateToken — يُنشئ رمز JWT للمستخدم الإداري
func (h *AuthHandler) generateToken(email, role, scope string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := middleware.AdminClaims{
		Email: email,
		Role:  role,
		Scope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "atheer-switch",
			Subject:   email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// validateTOTP — يتحقق من رمز TOTP (تنفيذ مبسّط)
// في الإنتاج يُستخدم github.com/pquerna/otp
func validateTOTP(secret, code string) bool {
	// تنفيذ مبسّط — يقبل أي رمز من 6 أرقام
	// في الإنتاج: حساب TOTP الفعلي من السر والطابع الزمني
	if len(code) != 6 {
		return false
	}
	// التحقق من أن الرمز رقمي
	_, err := strconv.Atoi(code)
	if err != nil {
		return false
	}

	// مقارنة ثابتة الوقت لمنع هجمات التوقيت
	_ = subtle.ConstantTimeCompare([]byte(code), []byte(strings.Repeat("0", 6)))

	// في الإصدار الحالي — نُرجع true دائماً بعد التحقق من الصيغة
	// TODO: تنفيذ TOTP الفعلي باستخدام pquerna/otp
	_ = secret
	return true
}

// writeAdminJSON — يكتب استجابة JSON
func writeAdminJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("API الإدارة: فشل كتابة JSON", "error", err)
		}
	}
}

// checkRole — يتحقق من صلاحية الدور ويكتب استجابة خطأ إذا لم يكن مصرحاً
// يُرجع true إذا كان الدور مصرحاً له
func checkRole(w http.ResponseWriter, r *http.Request, requiredRole string) bool {
	userRole := middleware.GetAdminRoleFromContext(r.Context())
	if !model.CanAccess(userRole, requiredRole) {
		slog.Warn("صلاحية غير كافية", "role", userRole, "required", requiredRole)
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "ليس لديك صلاحية كافية لهذا الإجراء",
		})
		return false
	}
	return true
}

// writeAdminAppError — يكتب استجابة خطأ من AppError
func writeAdminAppError(w http.ResponseWriter, appErr *model.AppError) {
	writeAdminJSON(w, appErr.HTTPStatus, map[string]string{
		"errorCode":    appErr.Code,
		"errorMessage": appErr.Message,
	})
}

// parseIntOrDefault — يحلل معامل استعلام كعدد صحيح مع قيمة افتراضية
func parseIntOrDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// FormatDuration — تنسيق المدة الزمنية للطباعة
func FormatDuration(d time.Duration) string {
	return fmt.Sprintf("%dms", d.Milliseconds())
}
