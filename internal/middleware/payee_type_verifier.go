// Modified for v3.0 Document Alignment
// حسب القسم 4 — الخطوة 5 — البند 4:
// السويتش يتحقق من PayeeType: يجلب SwitchRecord الخاص بـ ReceiverID
// يُقارن PayeeType (من الحزمة) بـ UserType (من السجل)
// إذا تعارَضا → رفض فوري
package middleware

import (
    "log/slog"
    "net/http"

    "github.com/atheer-payment/atheer-platform/internal/repository"
    "github.com/atheer-payment/atheer-platform/pkg/response"
)

type PayeeTypeVerifier struct {
    recordRepo *repository.SwitchRecordRepository
}

func NewPayeeTypeVerifier(repo *repository.SwitchRecordRepository) *PayeeTypeVerifier {
    return &PayeeTypeVerifier{recordRepo: repo}
}

func (v *PayeeTypeVerifier) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        packet := GetPayerPacket(r.Context())
        if packet == nil {
            response.BadRequest(w, response.ErrInternalError, "Packet not parsed")
            return
        }

        // جلب SwitchRecord الخاص بـ ReceiverID
        payeeRecord, err := v.recordRepo.GetByUserID(r.Context(), packet.ReceiverID)
        if err != nil || payeeRecord == nil {
            slog.Warn("PayeeType verification: ReceiverID not found",
                "receiverId", packet.ReceiverID)
            response.BadRequest(w, "ERR_PAYEE_TYPE_MISMATCH",
                "ReceiverID not registered in switch")
            return
        }

        // مقارنة PayeeType من الحزمة مع UserType من السجل
        if string(payeeRecord.UserType) != string(packet.PayeeType) {
            slog.Warn("PayeeType mismatch",
                "packet", packet.PayeeType,
                "record", payeeRecord.UserType)
            response.BadRequest(w, "ERR_PAYEE_TYPE_MISMATCH",
                "PayeeType does not match switch record")
            return
        }

        // تخزين payeeRecord في context للاستخدام لاحقاً
        ctx := SetPayeeRecord(r.Context(), payeeRecord)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
