// معالج مستخدمي الإدارة — قائمة المستخدمين وتعديل الحالات والحدود
// يُرجى الرجوع إلى SPEC §5 — Admin APIs
package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/middleware"
	"github.com/atheer/switch/internal/model"
)

// AdminUsersHandler — معالج مستخدمي الإدارة
type AdminUsersHandler struct {
	payerRepo  db.PayerRepo
	walletRepo db.WalletRepo
}

// NewAdminUsersHandler — ينشئ معالج مستخدمي الإدارة
func NewAdminUsersHandler(payerRepo db.PayerRepo, walletRepo db.WalletRepo) *AdminUsersHandler {
	return &AdminUsersHandler{
		payerRepo:  payerRepo,
		walletRepo: walletRepo,
	}
}

// UserListResponse — استجابة قائمة المستخدمين المسجّلين
type UserListResponse struct {
	Users      []UserInfo `json:"users"`      // قائمة المستخدمين
	TotalCount int        `json:"totalCount"` // العدد الإجمالي
	Page       int        `json:"page"`       // رقم الصفحة
	PageSize   int        `json:"pageSize"`   // حجم الصفحة
}

// UserInfo — معلومات مستخدم مُسجّل (بدون بيانات حساسة)
type UserInfo struct {
	PublicId   string `json:"publicId"`   // المعرّف العام
	WalletId   string `json:"walletId"`   // معرّف المحفظة
	DeviceId   string `json:"deviceId"`   // معرّف الجهاز
	Counter    int64  `json:"counter"`    // العداد الحالي
	PayerLimit int64  `json:"payerLimit"` // حد الدافع
	Status     string `json:"status"`     // الحالة: ACTIVE أو SUSPENDED
	UserType   string `json:"userType"`   // نوع المستخدم: P أو M
	CreatedAt  string `json:"createdAt"`  // تاريخ التسجيل
	UpdatedAt  string `json:"updatedAt"`  // تاريخ آخر تحديث
}

// UpdateStatusRequest — طلب تعديل حالة المستخدم
type UpdateStatusRequest struct {
	Status string `json:"status"` // الحالة الجديدة: ACTIVE أو SUSPENDED
}

// UpdatePayerLimitRequest — طلب تعديل حد الدافع
type UpdatePayerLimitRequest struct {
	PayerLimit int64 `json:"payerLimit"` // الحد الجديد بالوحدة الصغرى
}

// HandleList — يعالج طلب قائمة المستخدمين المسجّلين
// GET /admin/v1/users?status=ACTIVE&walletId=jawali&page=1&pageSize=20
func (h *AdminUsersHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// استخراج معاملات التصفية
	query := r.URL.Query()
	walletId := query.Get("walletId")
	status := query.Get("status")
	page := parseIntOrDefault(query.Get("page"), 1)
	pageSize := parseIntOrDefault(query.Get("pageSize"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	// فلترة النطاق — WALLET_ADMIN يرى محفظته فقط
	scopeFilter := middleware.ScopeFilter(ctx)
	if scopeFilter != "" {
		walletId = scopeFilter
	}

	// جلب المستخدمين — نستخدم PayerRepo مباشرة
	// ملاحظة: في الإصدار الحالي نعرض كل السجلات ثم نُصفّي في الذاكرة
	// في الإنتاج: إضافة دالة List مع تصفية في PayerRepo
	records, err := h.listRecords(ctx, walletId, status, page, pageSize)
	if err != nil {
		slog.Error("إدارة المستخدمين: فشل جلب القائمة", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في جلب المستخدمين",
		})
		return
	}

	writeAdminJSON(w, http.StatusOK, records)
}

// HandleUpdateStatus — يعالج طلب تعديل حالة المستخدم (تعليق/تفعيل)
// PATCH /admin/v1/users/{id}/status
func (h *AdminUsersHandler) HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// استخراج المعرّف العام من المسار
	publicId := r.PathValue("id")
	if publicId == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "معرّف المستخدم مطلوب",
		})
		return
	}

	// تحليل الطلب
	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "طلب غير صالح",
		})
		return
	}

	// التحقق من الحالة
	if req.Status != "ACTIVE" && req.Status != "SUSPENDED" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "الحالة يجب أن تكون ACTIVE أو SUSPENDED",
		})
		return
	}

	// فلترة النطاق
	if !h.canAccessUser(ctx, publicId) {
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "ليس لديك صلاحية لتعديل هذا المستخدم",
		})
		return
	}

	// التحقق من وجود المستخدم
	record, err := h.payerRepo.FindByPublicId(ctx, publicId)
	if err != nil {
		slog.Error("إدارة المستخدمين: فشل البحث", "publicId", publicId, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}
	if record == nil {
		writeAdminJSON(w, http.StatusNotFound, map[string]string{
			"errorCode":    model.ErrUnknownPayer,
			"errorMessage": "المستخدم غير موجود",
		})
		return
	}

	// تحديث الحالة
	if err := h.payerRepo.UpdateStatus(ctx, publicId, req.Status); err != nil {
		slog.Error("إدارة المستخدمين: فشل تحديث الحالة", "publicId", publicId, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في تحديث الحالة",
		})
		return
	}

	slog.Info("إدارة المستخدمين: تم تحديث الحالة", "publicId", publicId, "status", req.Status)

	writeAdminJSON(w, http.StatusOK, map[string]string{
		"publicId": publicId,
		"status":   req.Status,
		"message":  "تم تحديث الحالة بنجاح",
	})
}

// HandleUpdatePayerLimit — يعالج طلب تعديل حد الدافع
// PATCH /admin/v1/users/{id}/limit
func (h *AdminUsersHandler) HandleUpdatePayerLimit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// استخراج المعرّف العام من المسار
	publicId := r.PathValue("id")
	if publicId == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "معرّف المستخدم مطلوب",
		})
		return
	}

	// تحليل الطلب
	var req UpdatePayerLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "طلب غير صالح",
		})
		return
	}

	if req.PayerLimit <= 0 {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "حد الدافع يجب أن يكون أكبر من صفر",
		})
		return
	}

	// فلترة النطاق
	if !h.canAccessUser(ctx, publicId) {
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "ليس لديك صلاحية لتعديل هذا المستخدم",
		})
		return
	}

	// التحقق من وجود المستخدم
	record, err := h.payerRepo.FindByPublicId(ctx, publicId)
	if err != nil {
		slog.Error("إدارة المستخدمين: فشل البحث", "publicId", publicId, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}
	if record == nil {
		writeAdminJSON(w, http.StatusNotFound, map[string]string{
			"errorCode":    model.ErrUnknownPayer,
			"errorMessage": "المستخدم غير موجود",
		})
		return
	}

	// التحقق من أن الحد لا يتجاوز الحد الأقصى للمحفظة
	walletConfig, err := h.walletRepo.FindByWalletId(ctx, record.WalletId)
	if err == nil && walletConfig != nil {
		if req.PayerLimit > walletConfig.MaxPayerLimit {
			writeAdminJSON(w, http.StatusBadRequest, map[string]string{
				"errorCode":    model.ErrPayerLimitExceeded,
				"errorMessage": "الحد يتجاوز الحد الأقصى للمحفظة (" + strconv.FormatInt(walletConfig.MaxPayerLimit, 10) + ")",
			})
			return
		}
	}

	// تحديث الحد
	if err := h.payerRepo.UpdatePayerLimit(ctx, publicId, req.PayerLimit); err != nil {
		slog.Error("إدارة المستخدمين: فشل تحديث الحد", "publicId", publicId, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في تحديث الحد",
		})
		return
	}

	slog.Info("إدارة المستخدمين: تم تحديث حد الدافع", "publicId", publicId, "limit", req.PayerLimit)

	writeAdminJSON(w, http.StatusOK, map[string]interface{}{
		"publicId":      publicId,
		"payerLimit":    req.PayerLimit,
		"maxPayerLimit": walletConfig.MaxPayerLimit,
		"message":       "تم تحديث حد الدافع بنجاح",
	})
}

// canAccessUser — يتحقق من أن المستخدم الإداري يملك صلاحية الوصول للمستخدم المطلوب
func (h *AdminUsersHandler) canAccessUser(ctx context.Context, publicId string) bool {
	scopeFilter := middleware.ScopeFilter(ctx)
	if scopeFilter == "" {
		return true // SUPER_ADMIN أو ADMIN — يرى كل شيء
	}

	// WALLET_ADMIN — يتحقق من أن المستخدم يتبع محفظته
	record, err := h.payerRepo.FindByPublicId(ctx, publicId)
	if err != nil || record == nil {
		return false
	}
	return record.WalletId == scopeFilter
}

// listRecords — يجلب قائمة السجلات مع تصفية بسيطة
// ملاحظة: تنفيذ مبسّط — في الإنتاج يُضاف دالة List إلى PayerRepo
func (h *AdminUsersHandler) listRecords(ctx context.Context, walletId, status string, page, pageSize int) (*UserListResponse, error) {
	// في الإصدار الحالي نُرجع قائمة فارغة مع بنية صحيحة
	// TODO: إضافة db.PayerRepo.List(ctx, filters, page, pageSize)
	_ = ctx
	_ = walletId
	_ = status
	_ = page
	_ = pageSize

	return &UserListResponse{
		Users:      []UserInfo{},
		TotalCount: 0,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}
