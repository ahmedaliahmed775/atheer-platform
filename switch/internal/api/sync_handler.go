// معالج المزامنة — POST /api/v1/sync
// يُرجع العداد الأخير الصالح والحدود الحالية للدافع
// حسب العقد الموحد (OpenAPI): الحقول lastValidCounter, payerLimit, maxPayerLimit, status
package api

import (
        "log/slog"
        "net/http"

        "github.com/atheer/switch/internal/db"
        "github.com/atheer/switch/internal/model"
)

// SyncHandler — معالج مزامنة العداد والحدود
type SyncHandler struct {
        payerRepo  db.PayerRepo
        walletRepo db.WalletRepo
}

// NewSyncHandler — ينشئ معالج مزامنة جديد
func NewSyncHandler(payerRepo db.PayerRepo, walletRepo db.WalletRepo) *SyncHandler {
        return &SyncHandler{payerRepo: payerRepo, walletRepo: walletRepo}
}

// SyncRequest — طلب المزامنة (حسب العقد الموحد)
type SyncRequest struct {
        PublicId    string `json:"publicId"`              // المعرّف العام للدافع
        DeviceId    string `json:"deviceId"`              // معرّف الجهاز
        Timestamp   int64  `json:"timestamp"`             // الطابع الزمني بالثواني (Unix)
        RequestHmac string `json:"requestHmac,omitempty"` // HMAC اختياري — للمصادقة المتبادلة مستقبلاً
}

// SyncResponse — استجابة المزامنة (حسب العقد الموحد)
type SyncResponse struct {
        LastValidCounter int64  `json:"lastValidCounter"` // آخر عداد صالح (ليس counter)
        PayerLimit       int64  `json:"payerLimit"`       // حد الدافع بالوحدة الصغرى
        MaxPayerLimit    int64  `json:"maxPayerLimit"`    // الحد الأقصى المسموح لحد الدفع (من داشبورد السويتش لكل محفظة)
        Status           string `json:"status"`           // حالة الحساب: ACTIVE أو SUSPENDED أو REVOKED
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
        if req.DeviceId == "" {
                writeBadRequest(w, "deviceId مطلوب")
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

        // التحقق من مطابقة الجهاز
        if record.DeviceId != req.DeviceId {
                writeErrorWithCode(w, model.ErrDeviceMismatch)
                return
        }

        // البحث عن إعدادات المحفظة لمعرفة الحد الأقصى
        var maxPayerLimit int64
        walletCfg, err := h.walletRepo.FindByWalletId(ctx, record.WalletId)
        if err != nil || walletCfg == nil {
                // إذا لم نجد إعدادات المحفظة، نستخدم الحد الأقصى المسجل
                maxPayerLimit = record.PayerLimit
                slog.Warn("المزامنة: لم يتم العثور على إعدادات المحفظة، استخدام حد الدافع كحد أقصى",
                        "walletId", record.WalletId, "error", err)
        } else {
                maxPayerLimit = walletCfg.MaxPayerLimit
        }

        resp := SyncResponse{
                LastValidCounter: record.Counter,
                PayerLimit:       record.PayerLimit,
                MaxPayerLimit:    maxPayerLimit,
                Status:           record.Status,
        }

        slog.Debug("المزامنة: نجاح", "publicId", req.PublicId, "lastValidCounter", record.Counter)
        writeJSON(w, http.StatusOK, resp)
}
