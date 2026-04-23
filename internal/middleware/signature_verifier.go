// Modified for v3.0 Document Alignment
// SignatureVerifierA — التحقق من HMAC حسب القسم 4
// تم حذف SignatureVerifierB — Side B يُوثّق عبر mTLS لا HMAC
package middleware

import (
    "encoding/base64"
    "log/slog"
    "net/http"

    "github.com/atheer-payment/atheer-platform/internal/repository"
    "github.com/atheer-payment/atheer-platform/pkg/crypto"
    "github.com/atheer-payment/atheer-platform/pkg/response"
)

// SignatureVerifierA verifies Side A's HMAC signature
// Uses new v3.0 formula: LUK = HMAC-SHA256(Seed, Counter), Token = HMAC-SHA256(LUK, Amount||ReceiverID||PayeeType||WalletID||Counter)
type SignatureVerifierA struct {
    recordRepo *repository.SwitchRecordRepository
}

func NewSignatureVerifierA(recordRepo *repository.SwitchRecordRepository) *SignatureVerifierA {
    return &SignatureVerifierA{recordRepo: recordRepo}
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

        // 4. Verify HMAC using new v3.0 formula
        // WalletID comes from SwitchRecord (NOT from the packet — security requirement)
        err = crypto.VerifyTransactionHMAC(
            payerRecord.Seed,
            packet.Amount,
            packet.ReceiverID,
            string(packet.PayeeType),
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
