// معالج محافظ الإدارة — CRUD إعدادات المحافظ
// يُرجى الرجوع إلى SPEC §5 — Admin APIs
package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/middleware"
	"github.com/atheer/switch/internal/model"
)

// AdapterRegistry — واجهة سجل المحوّلات (لاستخدامها في اختبار الاتصال)
type AdapterRegistry interface {
	Get(walletId string) (model.WalletAdapter, error)
	List() []string
}

// AdminWalletsHandler — معالج محافظ الإدارة
type AdminWalletsHandler struct {
	walletRepo db.WalletRepo
	registry   AdapterRegistry
}

// NewAdminWalletsHandler — ينشئ معالج محافظ الإدارة
func NewAdminWalletsHandler(walletRepo db.WalletRepo, registry AdapterRegistry) *AdminWalletsHandler {
	return &AdminWalletsHandler{walletRepo: walletRepo, registry: registry}
}

// WalletListResponse — استجابة قائمة المحافظ
type WalletListResponse struct {
	Wallets []WalletInfo `json:"wallets"` // قائمة المحافظ
}

// WalletInfo — معلومات محفظة (البيانات الحساسة تظهر فقط لـ SUPER_ADMIN)
type WalletInfo struct {
	ID            int64  `json:"id"`            // المعرّف الداخلي
	WalletId      string `json:"walletId"`      // معرّف المحفظة
	BaseURL       string `json:"baseUrl"`       // عنوان API
	APIKey        string `json:"apiKey"`        // مفتاح API (SUPER_ADMIN فقط)
	Secret        string `json:"secret"`        // السر المشترك (SUPER_ADMIN فقط)
	MaxPayerLimit int64  `json:"maxPayerLimit"` // الحد الأقصى للدافع
	TimeoutMs     int    `json:"timeoutMs"`     // مهلة الطلب
	MaxRetries    int    `json:"maxRetries"`    // عدد إعادة المحاولات
	IsActive      bool   `json:"isActive"`      // هل المحفظة مفعّلة
	CreatedAt     string `json:"createdAt"`     // تاريخ الإنشاء
	UpdatedAt     string `json:"updatedAt"`     // تاريخ التحديث
}

// CreateWalletRequest — طلب إضافة محفظة جديدة
type CreateWalletRequest struct {
	WalletId      string `json:"walletId"`      // معرّف المحفظة
	BaseURL       string `json:"baseUrl"`       // عنوان API
	APIKey        string `json:"apiKey"`        // مفتاح API
	Secret        string `json:"secret"`        // السر المشترك
	MaxPayerLimit int64  `json:"maxPayerLimit"` // الحد الأقصى للدافع
	TimeoutMs     int    `json:"timeoutMs"`     // مهلة الطلب
	MaxRetries    int    `json:"maxRetries"`    // عدد إعادة المحاولات
}

// UpdateWalletRequest — طلب تحديث إعدادات محفظة
type UpdateWalletRequest struct {
	BaseURL       string `json:"baseUrl"`       // عنوان API
	APIKey        string `json:"apiKey"`        // مفتاح API
	Secret        string `json:"secret"`        // السر المشترك
	MaxPayerLimit int64  `json:"maxPayerLimit"` // الحد الأقصى للدافع
	TimeoutMs     int    `json:"timeoutMs"`     // مهلة الطلب
	MaxRetries    int    `json:"maxRetries"`    // عدد إعادة المحاولات
	IsActive      *bool  `json:"isActive"`      // هل المحفظة مفعّلة (مؤشر لتمييز عدم الإرسال)
}

// HandleList — يعالج طلب قائمة المحافظ
// GET /admin/v1/wallets
func (h *AdminWalletsHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	// التحقق من الصلاحية — ADMIN على الأقل
	if !checkRole(w, r, model.RoleAdmin) {
		return
	}

	ctx := r.Context()

	// فلترة النطاق — WALLET_ADMIN يرى محفظته فقط
	scopeFilter := middleware.ScopeFilter(ctx)

	wallets, err := h.walletRepo.List(ctx)
	if err != nil {
		slog.Error("إدارة المحافظ: فشل جلب القائمة", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في جلب المحافظ",
		})
		return
	}

	// تحويل إلى معلومات — SUPER_ADMIN يرى APIKey و Secret
	userRole := middleware.GetAdminRoleFromContext(ctx)
	showSecrets := userRole == model.RoleSuperAdmin

	var result []WalletInfo
	for _, wc := range wallets {
		// فلترة النطاق
		if scopeFilter != "" && wc.WalletId != scopeFilter {
			continue
		}

		info := WalletInfo{
			ID:            wc.ID,
			WalletId:      wc.WalletId,
			BaseURL:       wc.BaseURL,
			MaxPayerLimit: wc.MaxPayerLimit,
			TimeoutMs:     wc.TimeoutMs,
			MaxRetries:    wc.MaxRetries,
			IsActive:      wc.IsActive,
			CreatedAt:     wc.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:     wc.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		if showSecrets {
			info.APIKey = wc.APIKey
			info.Secret = wc.Secret
		}
		result = append(result, info)
	}

	if result == nil {
		result = []WalletInfo{}
	}

	writeAdminJSON(w, http.StatusOK, WalletListResponse{Wallets: result})
}

// HandleCreate — يعالج طلب إضافة محفظة جديدة
// POST /admin/v1/wallets
func (h *AdminWalletsHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	// التحقق من الصلاحية — SUPER_ADMIN فقط
	if !checkRole(w, r, model.RoleSuperAdmin) {
		return
	}

	ctx := r.Context()

	var req CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "طلب غير صالح",
		})
		return
	}

	// التحقق من الحقول المطلوبة
	if req.WalletId == "" || req.BaseURL == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "معرّف المحفظة وعنوان API مطلوبان",
		})
		return
	}

	// ضبط القيم الافتراضية
	if req.MaxPayerLimit <= 0 {
		req.MaxPayerLimit = 50000
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 10000
	}
	if req.MaxRetries <= 0 {
		req.MaxRetries = 2
	}

	// التحقق من عدم وجود محفظة بنفس المعرّف
	existing, err := h.walletRepo.FindByWalletId(ctx, req.WalletId)
	if err != nil {
		slog.Error("إدارة المحافظ: فشل البحث", "walletId", req.WalletId, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}
	if existing != nil {
		writeAdminJSON(w, http.StatusConflict, map[string]string{
			"errorCode":    model.ErrDeviceAlreadyRegistered,
			"errorMessage": "محفظة بنفس المعرّف موجودة مسبقاً",
		})
		return
	}

	// إنشاء المحفظة
	wc := &model.WalletConfig{
		WalletId:      req.WalletId,
		BaseURL:       req.BaseURL,
		APIKey:        req.APIKey,
		Secret:        req.Secret,
		MaxPayerLimit: req.MaxPayerLimit,
		TimeoutMs:     req.TimeoutMs,
		MaxRetries:    req.MaxRetries,
		IsActive:      true,
	}

	if err := h.walletRepo.Create(ctx, wc); err != nil {
		slog.Error("إدارة المحافظ: فشل الإنشاء", "walletId", req.WalletId, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في إنشاء المحفظة",
		})
		return
	}

	slog.Info("إدارة المحافظ: تم إنشاء محفظة جديدة", "walletId", req.WalletId)

	writeAdminJSON(w, http.StatusCreated, map[string]string{
		"walletId": req.WalletId,
		"message":  "تم إنشاء المحفظة بنجاح",
	})
}

// HandleUpdate — يعالج طلب تحديث إعدادات محفظة
// PUT /admin/v1/wallets/{id}
func (h *AdminWalletsHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	// التحقق من الصلاحية — SUPER_ADMIN فقط
	if !checkRole(w, r, model.RoleSuperAdmin) {
		return
	}

	ctx := r.Context()

	walletId := r.PathValue("id")
	if walletId == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "معرّف المحفظة مطلوب",
		})
		return
	}

	// فلترة النطاق
	scopeFilter := middleware.ScopeFilter(ctx)
	if scopeFilter != "" && scopeFilter != walletId {
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "ليس لديك صلاحية لتعديل هذه المحفظة",
		})
		return
	}

	// البحث عن المحفظة
	wc, err := h.walletRepo.FindByWalletId(ctx, walletId)
	if err != nil {
		slog.Error("إدارة المحافظ: فشل البحث", "walletId", walletId, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}
	if wc == nil {
		writeAdminJSON(w, http.StatusNotFound, map[string]string{
			"errorCode":    model.ErrWalletNotFound,
			"errorMessage": "المحفظة غير موجودة",
		})
		return
	}

	// تحليل الطلب
	var req UpdateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "طلب غير صالح",
		})
		return
	}

	// تحديث الحقول المُرسلة فقط
	if req.BaseURL != "" {
		wc.BaseURL = req.BaseURL
	}
	if req.APIKey != "" {
		wc.APIKey = req.APIKey
	}
	if req.Secret != "" {
		wc.Secret = req.Secret
	}
	if req.MaxPayerLimit > 0 {
		wc.MaxPayerLimit = req.MaxPayerLimit
	}
	if req.TimeoutMs > 0 {
		wc.TimeoutMs = req.TimeoutMs
	}
	if req.MaxRetries > 0 {
		wc.MaxRetries = req.MaxRetries
	}
	if req.IsActive != nil {
		wc.IsActive = *req.IsActive
	}

	if err := h.walletRepo.Update(ctx, wc); err != nil {
		slog.Error("إدارة المحافظ: فشل التحديث", "walletId", walletId, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في تحديث المحفظة",
		})
		return
	}

	slog.Info("إدارة المحافظ: تم تحديث المحفظة", "walletId", walletId)

	writeAdminJSON(w, http.StatusOK, map[string]string{
		"walletId": walletId,
		"message":  "تم تحديث المحفظة بنجاح",
	})
}

// HandlePatch — يعالج طلب تعديل جزئي لمحفظة (تفعيل/تعطيل)
// PATCH /admin/v1/wallets/{id}
func (h *AdminWalletsHandler) HandlePatch(w http.ResponseWriter, r *http.Request) {
	// إعادة توجيه إلى HandleUpdate — نفس المنطق
	h.HandleUpdate(w, r)
}

// HandleTest — يعالج طلب اختبار اتصال محفظة
// POST /admin/v1/wallets/{id}/test
func (h *AdminWalletsHandler) HandleTest(w http.ResponseWriter, r *http.Request) {
	// التحقق من الصلاحية — ADMIN على الأقل
	if !checkRole(w, r, model.RoleAdmin) {
		return
	}

	ctx := r.Context()

	walletId := r.PathValue("id")
	if walletId == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "معرّف المحفظة مطلوب",
		})
		return
	}

	// البحث عن المحفظة في قاعدة البيانات
	wc, err := h.walletRepo.FindByWalletId(ctx, walletId)
	if err != nil {
		slog.Error("إدارة المحافظ: فشل البحث", "walletId", walletId, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}
	if wc == nil {
		writeAdminJSON(w, http.StatusNotFound, map[string]string{
			"errorCode":    model.ErrWalletNotFound,
			"errorMessage": "المحفظة غير موجودة",
		})
		return
	}

	// محاولة الحصول على المحوّل من السجل
	adapter, err := h.registry.Get(walletId)
	if err != nil {
		writeAdminJSON(w, http.StatusOK, map[string]interface{}{
			"walletId": walletId,
			"status":   "not_registered",
			"message":  "المحوّل غير مسجّل في السويتش",
		})
		return
	}

	// اختبار الاتصال عبر VerifyAccessToken مع بيانات وهمية
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, verifyErr := adapter.VerifyAccessToken(testCtx, walletId, "test_connection_probe")

	writeAdminJSON(w, http.StatusOK, map[string]interface{}{
		"walletId": walletId,
		"status":   "reachable",
		"message":  "تم الاتصال بالمحفظة بنجاح",
		"verifyOk": verifyErr == nil,
	})
}

// formatWalletInfo — يحوّل WalletConfig إلى WalletInfo (البيانات الحساسة حسب الدور)
func formatWalletInfo(wc model.WalletConfig, showSecrets bool) WalletInfo {
	info := WalletInfo{
		ID:            wc.ID,
		WalletId:      wc.WalletId,
		BaseURL:       wc.BaseURL,
		MaxPayerLimit: wc.MaxPayerLimit,
		TimeoutMs:     wc.TimeoutMs,
		MaxRetries:    wc.MaxRetries,
		IsActive:      wc.IsActive,
		CreatedAt:     wc.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     wc.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if showSecrets {
		info.APIKey = wc.APIKey
		info.Secret = wc.Secret
	}
	return info
}

// parseInt64OrDefault — يحلل معامل كعدد int64 مع قيمة افتراضية
func parseInt64OrDefault(s string, def int64) int64 {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}
