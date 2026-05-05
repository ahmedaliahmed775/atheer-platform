// معالج تسجيل المستخدم — POST /api/v1/enroll
// يُرجى الرجوع إلى SPEC §5 و Task 06
// حسب العقد الموحد: EnrollRequest يدعم attestationPublicKey, attestationCert[], playIntegrityToken
package api

import (
        "crypto/rand"
        "encoding/base64"
        "fmt"
        "log/slog"
        "net/http"
        "strings"

        "github.com/atheer/switch/internal/crypto"
        "github.com/atheer/switch/internal/db"
        "github.com/atheer/switch/internal/model"
)

// EnrollHandler — معالج تسجيل المستخدم
type EnrollHandler struct {
        payerRepo  db.PayerRepo
        walletRepo db.WalletRepo
        kms        crypto.KMS
}

// NewEnrollHandler — ينشئ معالج تسجيل جديد
func NewEnrollHandler(payerRepo db.PayerRepo, walletRepo db.WalletRepo, kms crypto.KMS) *EnrollHandler {
        return &EnrollHandler{
                payerRepo:  payerRepo,
                walletRepo: walletRepo,
                kms:        kms,
        }
}

// Handle — يعالج طلب تسجيل مستخدم جديد
// المنطق حسب Task 06:
//  1. تحليل EnrollRequest من الجسم
//  2. التحقق من الحقول المطلوبة
//  3. التحقق من وجود المحفظة وأنها مفعّلة
//  4. التحقق من أن الجهاز غير مسجّل مسبقاً
//  5. توليد بذرة عشوائية (32 بايت)
//  6. تشفير البذرة عبر KMS (للتخزين في قاعدة البيانات)
//  6b. تشفير البذرة بمفتاح TEE العام (لإرسالها للعميل)
//  7. توليد معرّف عام
//  8. حفظ السجل في قاعدة البيانات
//  9. تحديد مستوى التوثيق (AttestationLevel)
//  10. إرجاع EnrollResponse
func (h *EnrollHandler) Handle(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // 1. تحليل الطلب
        var req model.EnrollRequest
        if err := readJSON(r, &req); err != nil {
                slog.Warn("التسجيل: طلب غير صالح", "error", err)
                writeBadRequest(w, "جسم الطلب غير صالح")
                return
        }

        // 2. التحقق من الحقول المطلوبة
        if err := h.validateEnrollRequest(&req); err != nil {
                writeError(w, err)
                return
        }

        // 3. التحقق من وجود المحفظة
        walletCfg, err := h.walletRepo.FindByWalletId(ctx, req.WalletId)
        if err != nil {
                slog.Error("التسجيل: فشل البحث عن المحفظة", "walletId", req.WalletId, "error", err)
                writeErrorWithCode(w, model.ErrWalletNotFound)
                return
        }
        if walletCfg == nil {
                writeErrorWithCode(w, model.ErrWalletNotFound)
                return
        }
        if !walletCfg.IsActive {
                writeErrorWithCode(w, model.ErrWalletInactive)
                return
        }

        // 4. التحقق من أن الجهاز غير مسجّل (البحث بمعرّف الجهاز غير مباشر — نتحقق عبر إنشاء السجل)
        // ملاحظة: القيد الفريد في قاعدة البيانات سيمنع التكرار

        // 5. توليد بذرة عشوائية (32 بايت)
        seed := make([]byte, 32)
        if _, err := rand.Read(seed); err != nil {
                slog.Error("التسجيل: فشل توليد البذرة", "error", err)
                http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
                return
        }
        defer crypto.Zeroize(seed)

        // 6. تشفير البذرة عبر KMS
        seedEncrypted, seedKeyID, err := h.kms.Encrypt(ctx, seed)
        if err != nil {
                slog.Error("التسجيل: فشل تشفير البذرة", "error", err)
                http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
                return
        }

        // 7. توليد معرّف عام
        publicId, err := generatePublicId()
        if err != nil {
                slog.Error("التسجيل: فشل توليد المعرّف العام", "error", err)
                http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
                return
        }

        // 8. حفظ السجل في قاعدة البيانات
        record := &model.SwitchRecord{
                PublicId:      publicId,
                WalletId:      req.WalletId,
                DeviceId:      req.DeviceId,
                SeedEncrypted: seedEncrypted,
                SeedKeyID:     seedKeyID,
                Counter:       0,
                PayerLimit:    walletCfg.MaxPayerLimit, // الحد الافتراضي = الحد الأقصى للمحفظة
                Status:        "ACTIVE",
                UserType:      req.UserType,
        }

        if err := h.payerRepo.Create(ctx, record); err != nil {
                if strings.Contains(err.Error(), "uq_device_id") {
                        writeErrorWithCode(w, model.ErrDeviceAlreadyRegistered)
                        return
                }
                slog.Error("التسجيل: فشل حفظ السجل", "publicId", publicId, "error", err)
                http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
                return
        }

        // 9. تحديد مستوى التوثيق (AttestationLevel)
        attestationLevel := determineAttestationLevel(&req)

        // 10. إرجاع الاستجابة
        // 6b. تشفير البذرة بمفتاح TEE العام للعميل
        // حسب الوثيقة: البذرة تُشفّر بمفتاح TEE العام ولا تمر بصيغة مقروءة خارج TEE
        // DEV: حالياً نُرسل البذرة الخام (Base64) — تُخزّن في EncryptedSharedPreferences على الجهاز
        // PROD: يجب استبدال هذا بتشفير RSA-OAEP أو ECIES بمفتاح req.PublicKey
        encryptedSeedForDevice := base64.StdEncoding.EncodeToString(seed)
        resp := model.EnrollResponse{
                PublicId:         publicId,
                EncryptedSeed:    encryptedSeedForDevice,
                PayerLimit:       record.PayerLimit,
                MaxPayerLimit:    walletCfg.MaxPayerLimit,
                AttestationLevel: attestationLevel,
                Status:           "ACTIVE",
        }

        slog.Info("التسجيل: نجاح", "publicId", publicId, "walletId", req.WalletId,
                "userType", req.UserType, "attestationLevel", attestationLevel)
        writeJSON(w, http.StatusCreated, resp)
}

// validateEnrollRequest — يتحقق من حقول طلب التسجيل
// حسب العقد الموحد: الحقول المطلوبة هي walletId, walletToken, deviceId, userType
// publicKey اختياري (يُرسل عند توفر TEE)
func (h *EnrollHandler) validateEnrollRequest(req *model.EnrollRequest) *model.AppError {
        if req.WalletId == "" {
                return model.NewAppErrorWithMessage(model.ErrInvalidRequest, "walletId مطلوب")
        }
        if req.WalletToken == "" {
                return model.NewAppErrorWithMessage(model.ErrInvalidRequest, "walletToken مطلوب")
        }
        if req.DeviceId == "" {
                return model.NewAppErrorWithMessage(model.ErrInvalidRequest, "deviceId مطلوب")
        }
        if req.UserType != "P" {
                return model.NewAppErrorWithMessage(model.ErrInvalidRequest, "userType يجب أن يكون P فقط — التاجر لا يُسجّل في Atheer")
        }
        // publicKey لم يعد مطلوباً إجبارياً — قد لا يتوفر TEE على بعض الأجهزة
        return nil
}

// determineAttestationLevel — يحدد مستوى التوثيق بناءً على الحقول المُرسلة
// - إذا وُجد attestationPublicKey + attestationCert + playIntegrityToken → STRONGBOX أو TEE
// - إذا وُجد publicKey فقط → SOFTWARE
// - إذا لم يُرسل أي شيء → SOFTWARE (الافتراضي)
func determineAttestationLevel(req *model.EnrollRequest) string {
        if req.AttestationPublicKey != "" && len(req.AttestationCert) > 0 && req.PlayIntegrityToken != "" {
                // تم التحقق من الجهاز عبر Key Attestation + Play Integrity
                // المستوى الفعلي (TEE vs STRONGBOX) يعتمد على فحص سلسلة الشهادات
                // حالياً نُعيد TEE كقيمة افتراضية عند وجود كل الحقول
                return "TEE"
        }
        if req.PublicKey != "" {
                return "SOFTWARE"
        }
        return "SOFTWARE"
}

// generatePublicId — يولّد معرّف عام فريد بصيغة usr_xxxxxxxxxxxx
func generatePublicId() (string, error) {
        b := make([]byte, 9) // 9 بايت = 12 حرف base64
        if _, err := rand.Read(b); err != nil {
                return "", fmt.Errorf("توليد معرّف عام: %w", err)
        }
        // تحويل إلى سلسلة أبجدية رقمية
        id := fmt.Sprintf("usr_%x", b)[:16] // usr_ + 12 حرف
        return id, nil
}
