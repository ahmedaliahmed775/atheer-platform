// Modified for v3.0 Document Alignment
// SignatureVerifierA — التحقق من HMAC حسب القسم 4
// تم حذف SignatureVerifierB — Side B يُوثّق عبر mTLS لا HMAC
//
// v3.0 Security Fix:
// - PayeeType تم حذفه من HMAC — السويتش يحدد UserType تلقائياً
// - Currency أُضيف للحماية من هجمات تبديل العملة
// - Seed يتم فك تشفيره عبر KMS قبل التحقق من HMAC
package middleware

import (
    "encoding/base64"
    "log/slog"
    "net/http"

    "github.com/atheer-payment/atheer-platform/internal/repository"
    "github.com/atheer-payment/atheer-platform/internal/service"
    "github.com/atheer-payment/atheer-platform/pkg/crypto"
    "github.com/atheer-payment/atheer-platform/pkg/response"
)

// SignatureVerifierA verifies Side A's HMAC signature
// Uses v3.0 formula: LUK = HMAC-SHA256(Seed, Counter), Token = HMAC-SHA256(LUK, Amount||ReceiverID||Currency||WalletID||Counter)
type SignatureVerifierA struct {
    recordRepo *repository.SwitchRecordRepository
    kms        service.KMSProvider
}

func NewSignatureVerifierA(recordRepo *repository.SwitchRecordRepository, kms service.KMSProvider) *SignatureVerifierA {
    return &SignatureVerifierA{
        recordRepo: recordRepo,
        kms:        kms,
    }
}

func (sv *SignatureVerifierA) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        packet := GetPayerPacket(r.Context())
        if packet == nil {
            response.BadRequest(w, response.ErrInternalError, "Packet not parsed")
            return
        }

        // 1. Get payer SwitchRecord by PublicID
        payerRecord, err := sv.recordRepo.GetByPublicID(r.Context(), packet.PublicID)
        if err != nil || payerRecord == nil {
            response.Forbidden(w, "ERR_HMAC_MISMATCH", "Payer not found in switch")
            return
        }

        // 2. Verify payer is active
        if payerRecord.Status != "ACTIVE" {
            response.Forbidden(w, "ERR_HMAC_MISMATCH", "Payer account is "+payerRecord.Status)
            return
        }

        // 3. Decode HMAC bytes
        hmacBytes := packet.HMAC
        if len(hmacBytes) == 0 {
            response.BadRequest(w, "ERR_HMAC_MISMATCH", "HMAC is empty")
            return
        }

        // If HMAC is base64-encoded string in JSON, try to decode
        if decoded, err := base64.StdEncoding.DecodeString(string(hmacBytes)); err == nil && len(decoded) > 0 {
            hmacBytes = decoded
        }

        // 4. Decrypt Seed via KMS before HMAC verification
        // Seed مشفّر في قاعدة البيانات — يجب فك التشفير قبل الاستخدام
        decryptedSeed, err := sv.kms.Decrypt(payerRecord.Seed)
        if err != nil {
            slog.Error("Failed to decrypt seed via KMS — REJECTING (fail-closed)",
                "publicId", packet.PublicID,
                "error", err,
            )
            response.ServiceUnavailable(w, "ERR_KMS", "Security service unavailable — retry later")
            return
        }

        // 5. Verify HMAC using v3.0 formula
        // WalletID comes from SwitchRecord (NOT from the packet — security requirement)
        // Currency comes from the packet (part of HMAC to prevent currency swap attacks)
        err = crypto.VerifyTransactionHMAC(
            decryptedSeed,
            packet.Amount,
            packet.ReceiverID,
            packet.Currency,
            payerRecord.WalletID, // من السجل — ليس من الحزمة
            packet.Counter,
            hmacBytes,
        )
        if err != nil {
            slog.Warn("HMAC signature verification failed",
                "publicId", packet.PublicID,
                "counter", packet.Counter,
                "error", err,
            )
            response.Forbidden(w, "ERR_HMAC_MISMATCH", "HMAC signature invalid")
            return
        }

        slog.Debug("HMAC signature verified",
            "publicId", packet.PublicID,
            "counter", packet.Counter,
        )

        // Store payer record in context
        ctx := SetPayerRecord(r.Context(), payerRecord)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
