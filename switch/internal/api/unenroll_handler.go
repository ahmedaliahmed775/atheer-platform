// معالج إلغاء التسجيل — POST /api/v1/unenroll
// يحذف سجل الدافع من قاعدة البيانات (إلغاء تسجيل الجهاز)
package api

import (
	"log/slog"
	"net/http"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/model"
)

// UnenrollHandler — معالج إلغاء التسجيل
type UnenrollHandler struct {
	payerRepo db.PayerRepo
}

// NewUnenrollHandler — ينشئ معالج إلغاء التسجيل
func NewUnenrollHandler(payerRepo db.PayerRepo) *UnenrollHandler {
	return &UnenrollHandler{payerRepo: payerRepo}
}

// UnenrollRequest — طلب إلغاء التسجيل
type UnenrollRequest struct {
	PublicId string `json:"publicId"` // المعرّف العام للدافع
	DeviceId string `json:"deviceId"` // معرّف الجهاز
}

// UnenrollResponse — استجابة إلغاء التسجيل
type UnenrollResponse struct {
	PublicId string `json:"publicId"` // المعرّف العام
	Status   string `json:"status"`   // الحالة: UNENROLLED
}

// Handle — يعالج طلب إلغاء التسجيل
func (h *UnenrollHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req UnenrollRequest
	if err := readJSON(r, &req); err != nil {
		writeBadRequest(w, "جسم الطلب غير صالح")
		return
	}

	if req.PublicId == "" {
		writeBadRequest(w, "publicId مطلوب")
		return
	}

	// التحقق من وجود السجل
	record, err := h.payerRepo.FindByPublicId(ctx, req.PublicId)
	if err != nil {
		slog.Error("إلغاء التسجيل: فشل البحث", "publicId", req.PublicId, "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}
	if record == nil {
		writeErrorWithCode(w, model.ErrUnknownPayer)
		return
	}

	// التحقق من مطابقة الجهاز (أمان إضافي)
	if req.DeviceId != "" && record.DeviceId != req.DeviceId {
		writeErrorWithCode(w, model.ErrDeviceMismatch)
		return
	}

	// حذف السجل
	if err := h.payerRepo.Delete(ctx, req.PublicId); err != nil {
		slog.Error("إلغاء التسجيل: فشل الحذف", "publicId", req.PublicId, "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}

	resp := UnenrollResponse{
		PublicId: req.PublicId,
		Status:   "UNENROLLED",
	}

	slog.Info("إلغاء التسجيل: نجاح", "publicId", req.PublicId)
	writeJSON(w, http.StatusOK, resp)
}
