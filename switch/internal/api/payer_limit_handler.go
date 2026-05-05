// معالج تعديل حد الدافع — POST /api/v1/payer-limit
// يسمح بتعديل حد الدافع بشرط ألا يتجاوز الحد الأقصى
// حسب العقد الموحد: الطلب يرسل newLimit وليس payerLimit
package api

import (
	"log/slog"
	"net/http"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/model"
)

// PayerLimitHandler — معالج تعديل حد الدافع
type PayerLimitHandler struct {
	payerRepo  db.PayerRepo
	walletRepo db.WalletRepo
}

// NewPayerLimitHandler — ينشئ معالج تعديل حد الدافع
func NewPayerLimitHandler(payerRepo db.PayerRepo, walletRepo db.WalletRepo) *PayerLimitHandler {
	return &PayerLimitHandler{
		payerRepo:  payerRepo,
		walletRepo: walletRepo,
	}
}

// PayerLimitRequest — طلب تعديل حد الدافع (حسب العقد الموحد)
type PayerLimitRequest struct {
	PublicId    string `json:"publicId"`              // المعرّف العام للدافع
	DeviceId    string `json:"deviceId"`              // معرّف الجهاز
	NewLimit    int64  `json:"newLimit"`              // الحد الجديد بالوحدة الصغرى
	Timestamp   int64  `json:"timestamp"`             // الطابع الزمني بالثواني (Unix)
	RequestHmac string `json:"requestHmac,omitempty"` // HMAC اختياري — للمصادقة المتبادلة مستقبلاً
}

// PayerLimitResponse — استجابة تعديل حد الدافع (حسب العقد الموحد)
type PayerLimitResponse struct {
	PublicId        string `json:"publicId"`        // المعرّف العام
	PayerLimit      int64  `json:"payerLimit"`      // الحد الجديد بعد التحديث
	MaxAllowedLimit int64  `json:"maxAllowedLimit"` // الحد الأقصى المسموح للمحفظة
	Status          string `json:"status"`          // حالة الحساب: ACTIVE
}

// Handle — يعالج طلب تعديل حد الدافع
func (h *PayerLimitHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req PayerLimitRequest
	if err := readJSON(r, &req); err != nil {
		writeBadRequest(w, "جسم الطلب غير صالح")
		return
	}

	if req.PublicId == "" {
		writeBadRequest(w, "publicId مطلوب")
		return
	}
	if req.DeviceId == "" {
		writeBadRequest(w, "deviceId مطلوب")
		return
	}
	if req.NewLimit <= 0 {
		writeBadRequest(w, "newLimit يجب أن يكون أكبر من صفر")
		return
	}

	// البحث عن السجل
	record, err := h.payerRepo.FindByPublicId(ctx, req.PublicId)
	if err != nil {
		slog.Error("حد الدافع: فشل البحث", "publicId", req.PublicId, "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}
	if record == nil {
		writeErrorWithCode(w, model.ErrUnknownPayer)
		return
	}

	// التحقق من مطابقة الجهاز
	if record.DeviceId != req.DeviceId {
		writeErrorWithCode(w, model.ErrDeviceMismatch)
		return
	}

	// البحث عن إعدادات المحفظة لمعرفة الحد الأقصى
	walletCfg, err := h.walletRepo.FindByWalletId(ctx, record.WalletId)
	if err != nil {
		slog.Error("حد الدافع: فشل البحث عن المحفظة", "walletId", record.WalletId, "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}
	if walletCfg == nil {
		writeErrorWithCode(w, model.ErrWalletNotFound)
		return
	}

	// التحقق من أن الحد الجديد لا يتجاوز الحد الأقصى
	if req.NewLimit > walletCfg.MaxPayerLimit {
		writeErrorWithCode(w, model.ErrPayerLimitExceeded)
		return
	}

	// تحديث الحد
	if err := h.payerRepo.UpdatePayerLimit(ctx, req.PublicId, req.NewLimit); err != nil {
		slog.Error("حد الدافع: فشل التحديث", "publicId", req.PublicId, "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}

	resp := PayerLimitResponse{
		PublicId:        req.PublicId,
		PayerLimit:      req.NewLimit,
		MaxAllowedLimit: walletCfg.MaxPayerLimit,
		Status:          record.Status,
	}

	slog.Info("حد الدافع: تم التحديث",
		"publicId", req.PublicId,
		"newLimit", req.NewLimit,
		"maxAllowedLimit", walletCfg.MaxPayerLimit,
	)
	writeJSON(w, http.StatusOK, resp)
}
