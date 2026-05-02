// معالج تعديل حد الدافع — POST /api/v1/payer-limit
// يسمح بتعديل حد الدافع بشرط ألا يتجاوز الحد الأقصى
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

// PayerLimitRequest — طلب تعديل حد الدافع
type PayerLimitRequest struct {
	PublicId   string `json:"publicId"`   // المعرّف العام للدافع
	PayerLimit int64  `json:"payerLimit"` // الحد الجديد بالوحدة الصغرى
}

// PayerLimitResponse — استجابة تعديل حد الدافع
type PayerLimitResponse struct {
	PublicId      string `json:"publicId"`      // المعرّف العام
	PayerLimit    int64  `json:"payerLimit"`     // الحد الجديد
	MaxPayerLimit int64  `json:"maxPayerLimit"`  // الحد الأقصى المسموح
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
	if req.PayerLimit <= 0 {
		writeBadRequest(w, "payerLimit يجب أن يكون أكبر من صفر")
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
	if req.PayerLimit > walletCfg.MaxPayerLimit {
		writeBadRequest(w, "payerLimit يتجاوز الحد الأقصى المسموح")
		return
	}

	// تحديث الحد
	if err := h.payerRepo.UpdatePayerLimit(ctx, req.PublicId, req.PayerLimit); err != nil {
		slog.Error("حد الدافع: فشل التحديث", "publicId", req.PublicId, "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}

	resp := PayerLimitResponse{
		PublicId:      req.PublicId,
		PayerLimit:    req.PayerLimit,
		MaxPayerLimit: walletCfg.MaxPayerLimit,
	}

	slog.Info("حد الدافع: تم التحديث",
		"publicId", req.PublicId,
		"newLimit", req.PayerLimit,
		"maxLimit", walletCfg.MaxPayerLimit,
	)
	writeJSON(w, http.StatusOK, resp)
}
