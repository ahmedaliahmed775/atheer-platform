// معالج حسابات الإدارة — CRUD لمستخدمي الداشبورد
// يُرجى الرجوع إلى SPEC §5 — Admin APIs
package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/middleware"
	"github.com/atheer/switch/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// AdminAdminsHandler — معالج حسابات الإدارة
type AdminAdminsHandler struct {
	adminRepo db.AdminRepo
}

// NewAdminAdminsHandler — ينشئ معالج حسابات الإدارة
func NewAdminAdminsHandler(adminRepo db.AdminRepo) *AdminAdminsHandler {
	return &AdminAdminsHandler{adminRepo: adminRepo}
}

// AdminListResponse — استجابة قائمة المستخدمين الإداريين
type AdminListResponse struct {
	Admins []AdminInfo `json:"admins"` // قائمة المستخدمين الإداريين
}

// AdminInfo — معلومات مستخدم إداري (بدون كلمة المرور)
type AdminInfo struct {
	ID          int64  `json:"id"`          // المعرّف الداخلي
	Email       string `json:"email"`       // البريد الإلكتروني
	Role        string `json:"role"`        // الدور
	Scope       string `json:"scope"`       // نطاق الصلاحيات
	IsActive    bool   `json:"isActive"`    // هل الحساب مفعّل
	LastLoginAt string `json:"lastLoginAt"` // آخر تسجيل دخول
	CreatedAt   string `json:"createdAt"`   // تاريخ الإنشاء
	UpdatedAt   string `json:"updatedAt"`   // تاريخ التحديث
}

// CreateAdminRequest — طلب إضافة مستخدم إداري جديد
type CreateAdminRequest struct {
	Email    string `json:"email"`    // البريد الإلكتروني
	Password string `json:"password"` // كلمة المرور
	Role     string `json:"role"`     // الدور
	Scope    string `json:"scope"`    // نطاق الصلاحيات
}

// PatchAdminRequest — طلب تعديل جزئي لمستخدم إداري
type PatchAdminRequest struct {
	IsActive *bool   `json:"isActive"` // تفعيل/تعطيل (مؤشر لتمييز عدم الإرسال)
	Role     *string `json:"role"`     // الدور (مؤشر لتمييز عدم الإرسال)
	Scope    *string `json:"scope"`    // النطاق (مؤشر لتمييز عدم الإرسال)
}

// HandleList — يعالج طلب قائمة المستخدمين الإداريين
// GET /admin/v1/admins
func (h *AdminAdminsHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// التحقق من الصلاحية — ADMIN على الأقل
	userRole := middleware.GetAdminRoleFromContext(ctx)
	if !model.CanAccess(userRole, model.RoleAdmin) {
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "ليس لديك صلاحية لعرض المستخدمين الإداريين",
		})
		return
	}

	users, err := h.adminRepo.List(ctx)
	if err != nil {
		slog.Error("إدارة الحسابات: فشل جلب القائمة", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في جلب المستخدمين الإداريين",
		})
		return
	}

	// تحويل إلى معلومات عامة (بدون كلمة المرور)
	var result []AdminInfo
	for _, u := range users {
		result = append(result, adminToInfo(u))
	}

	if result == nil {
		result = []AdminInfo{}
	}

	writeAdminJSON(w, http.StatusOK, AdminListResponse{Admins: result})
}

// HandleCreate — يعالج طلب إضافة مستخدم إداري جديد
// POST /admin/v1/admins
func (h *AdminAdminsHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// التحقق من الصلاحية — SUPER_ADMIN فقط
	userRole := middleware.GetAdminRoleFromContext(ctx)
	if !model.CanAccess(userRole, model.RoleSuperAdmin) {
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "فقط المدير الأعلى يمكنه إنشاء مستخدمين إداريين",
		})
		return
	}

	var req CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "طلب غير صالح",
		})
		return
	}

	// التحقق من الحقول المطلوبة
	if req.Email == "" || req.Password == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "البريد الإلكتروني وكلمة المرور مطلوبان",
		})
		return
	}

	// التحقق من صحة الدور
	validRoles := map[string]bool{
		model.RoleSuperAdmin:  true,
		model.RoleAdmin:       true,
		model.RoleWalletAdmin: true,
		model.RoleViewer:      true,
	}
	if !validRoles[req.Role] {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "دور غير صالح",
		})
		return
	}

	// ضبط القيم الافتراضية
	if req.Scope == "" {
		req.Scope = "global"
	}

	// التحقق من عدم وجود مستخدم بنفس البريد
	existing, err := h.adminRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		slog.Error("إدارة الحسابات: فشل البحث", "email", req.Email, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}
	if existing != nil {
		writeAdminJSON(w, http.StatusConflict, map[string]string{
			"errorCode":    "EMAIL_EXISTS",
			"errorMessage": "بريد إلكتروني مسجّل مسبقاً",
		})
		return
	}

	// تجزئة كلمة المرور
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("إدارة الحسابات: فشل تجزئة كلمة المرور", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في إنشاء الحساب",
		})
		return
	}

	// إنشاء المستخدم
	user := &model.AdminUser{
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
		Scope:        req.Scope,
		IsActive:     true,
	}

	if err := h.adminRepo.Create(ctx, user); err != nil {
		slog.Error("إدارة الحسابات: فشل الإنشاء", "email", req.Email, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في إنشاء الحساب",
		})
		return
	}

	slog.Info("إدارة الحسابات: تم إنشاء مستخدم إداري", "email", req.Email, "role", req.Role)

	writeAdminJSON(w, http.StatusCreated, map[string]string{
		"email":   req.Email,
		"message": "تم إنشاء الحساب بنجاح",
	})
}

// HandlePatch — يعالج طلب تعديل جزئي لمستخدم إداري
// PATCH /admin/v1/admins/{id}
func (h *AdminAdminsHandler) HandlePatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// التحقق من الصلاحية — SUPER_ADMIN فقط
	userRole := middleware.GetAdminRoleFromContext(ctx)
	if !model.CanAccess(userRole, model.RoleSuperAdmin) {
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"errorCode":    model.ErrForbiddenRole,
			"errorMessage": "فقط المدير الأعلى يمكنه تعديل المستخدمين الإداريين",
		})
		return
	}

	// استخراج المعرّف من المسار
	idStr := r.PathValue("id")
	if idStr == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "معرّف المستخدم مطلوب",
		})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "معرّف المستخدم غير صالح",
		})
		return
	}

	// البحث عن المستخدم
	user, err := h.adminRepo.FindByID(ctx, id)
	if err != nil {
		slog.Error("إدارة الحسابات: فشل البحث", "id", id, "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ داخلي",
		})
		return
	}
	if user == nil {
		writeAdminJSON(w, http.StatusNotFound, map[string]string{
			"errorCode":    "ADMIN_NOT_FOUND",
			"errorMessage": "المستخدم الإداري غير موجود",
		})
		return
	}

	// تحليل الطلب
	var req PatchAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"errorCode":    model.ErrInvalidRequest,
			"errorMessage": "طلب غير صالح",
		})
		return
	}

	// تعديل حالة التفعيل
	if req.IsActive != nil {
		if err := h.adminRepo.UpdateStatus(ctx, id, *req.IsActive); err != nil {
			slog.Error("إدارة الحسابات: فشل تحديث الحالة", "id", id, "error", err)
			writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
				"errorCode":    "INTERNAL_ERROR",
				"errorMessage": "خطأ في تحديث الحالة",
			})
			return
		}
		user.IsActive = *req.IsActive
	}

	// تعديل الدور أو النطاق
	if req.Role != nil || req.Scope != nil {
		if req.Role != nil {
			validRoles := map[string]bool{
				model.RoleSuperAdmin:  true,
				model.RoleAdmin:       true,
				model.RoleWalletAdmin: true,
				model.RoleViewer:      true,
			}
			if !validRoles[*req.Role] {
				writeAdminJSON(w, http.StatusBadRequest, map[string]string{
					"errorCode":    model.ErrInvalidRequest,
					"errorMessage": "دور غير صالح",
				})
				return
			}
			user.Role = *req.Role
		}
		if req.Scope != nil {
			user.Scope = *req.Scope
		}
		if err := h.adminRepo.Update(ctx, user); err != nil {
			slog.Error("إدارة الحسابات: فشل التحديث", "id", id, "error", err)
			writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
				"errorCode":    "INTERNAL_ERROR",
				"errorMessage": "خطأ في تحديث المستخدم",
			})
			return
		}
	}

	slog.Info("إدارة الحسابات: تم تحديث مستخدم إداري", "id", id)

	// إعادة قراءة المستخدم المحدّث
	updated, _ := h.adminRepo.FindByID(ctx, id)
	if updated != nil {
		writeAdminJSON(w, http.StatusOK, adminToInfo(*updated))
		return
	}

	writeAdminJSON(w, http.StatusOK, map[string]string{
		"id":      idStr,
		"message": "تم التحديث بنجاح",
	})
}

// adminToInfo — يحوّل AdminUser إلى AdminInfo بدون بيانات حساسة
func adminToInfo(u model.AdminUser) AdminInfo {
	lastLogin := ""
	if u.LastLoginAt != nil {
		lastLogin = u.LastLoginAt.Format("2006-01-02 15:04:05")
	}

	return AdminInfo{
		ID:          u.ID,
		Email:       u.Email,
		Role:        u.Role,
		Scope:       u.Scope,
		IsActive:    u.IsActive,
		LastLoginAt: lastLogin,
		CreatedAt:   u.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   u.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
