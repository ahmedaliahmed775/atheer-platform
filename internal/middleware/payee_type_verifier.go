// Modified for v3.0 Document Alignment
// حسب القسم 4 — الخطوة 5 — البند 4:
// السويتش يحدد UserType تلقائياً من SwitchRecord
// لم يعد يعتمد على PayeeType من الحزمة — السويتش هو المصدر الوحيد للحقيقة
//
// هذا الـ middleware يتحقق أن المستقبل مسجل في السويتش
// ويخزّن سجل المستقبل في context لاستخدامه في تحديد TransactionType
package middleware

import (
    "log/slog"
    "net/http"

    "github.com/atheer-payment/atheer-platform/internal/repository"
    "github.com/atheer-payment/atheer-platform/pkg/response"
)

// PayeeTypeVerifier يتحقق من أن المستقبل مسجل في السويتش
// ويخزّن سجل المستقبل في context لتحديد TransactionType لاحقاً
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
        // السويتش يحدد UserType من السجل — لا من الحزمة
        payeeRecord, err := v.recordRepo.GetByUserID(r.Context(), packet.ReceiverID)
        if err != nil || payeeRecord == nil {
            slog.Warn("Payee verification: ReceiverID not found",
                "receiverId", packet.ReceiverID)
            response.BadRequest(w, "ERR_PAYEE_NOT_FOUND",
                "ReceiverID not registered in switch")
            return
        }

        // التحقق من أن حساب المستقبل نشط
        if payeeRecord.Status != "ACTIVE" {
            slog.Warn("Payee account not active",
                "receiverId", packet.ReceiverID,
                "status", payeeRecord.Status)
            response.BadRequest(w, "ERR_PAYEE_SUSPENDED",
                "Receiver account is "+payeeRecord.Status)
            return
        }

        slog.Debug("Payee verified from SwitchRecord",
            "receiverId", packet.ReceiverID,
            "payeeUserType", payeeRecord.UserType,
        )

        // تخزين payeeRecord في context لاستخدامه في TransactionTypeResolver
        ctx := SetPayeeRecord(r.Context(), payeeRecord)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
