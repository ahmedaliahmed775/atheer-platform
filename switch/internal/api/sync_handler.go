// معالج المزامنة — POST /api/v1/sync
// يُرجع العداد الحالي وحد الدافع للمستخدم
package api

import (
	"log/slog"
	"net/http"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/model"
)

// SyncHandler — معالج مزامنة العداد والحدود
type SyncHandler struct {
	payerRepo db.PayerRepo
}

// NewSyncHandler — ينشئ معالج مزامنة جديد
func NewSyncHandler(payerRepo db.PayerRepo) *SyncHandler {
	return &SyncHandler{payerRepo: payerRepo}
}

// SyncRequest — طلب المزامنة
type SyncRequest struct {
	PublicId string `json:"publicId"` // المعرّف العام للدافع
}

// SyncResponse — استجابة المزامنة
type SyncResponse struct {
	PublicId   string `json:"publicId"`    // المعرّف العام
	Counter    int64  `json:"counter"`     // العداد الحالي
	PayerLimit int64  `json:"payerLimit"`  // حد الدافع
	Status     string `json:"status"`      // حالة السجل
}

// Handle — يعالج طلب مزامنة العداد والحدود
func (h *SyncHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SyncRequest
	if err := readJSON(r, &req); err != nil {
		writeBadRequest(w, "جسم الطلب غير صالح")
		return
	}

	if req.PublicId == "" {
		writeBadRequest(w, "publicId مطلوب")
		return
	}

	// البحث عن السجل
	record, err := h.payerRepo.FindByPublicId(ctx, req.PublicId)
	if err != nil {
		slog.Error("المزامنة: فشل البحث", "publicId", req.PublicId, "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}
	if record == nil {
		writeErrorWithCode(w, model.ErrUnknownPayer)
		return
	}

	resp := SyncResponse{
		PublicId:   record.PublicId,
		Counter:    record.Counter,
		PayerLimit: record.PayerLimit,
		Status:     record.Status,
	}

	slog.Debug("المزامنة: نجاح", "publicId", req.PublicId, "counter", record.Counter)
	writeJSON(w, http.StatusOK, resp)
}
