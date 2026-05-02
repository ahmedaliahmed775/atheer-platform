// طبقة البوابة (GATE) — الطبقة الأولى من خط معالجة المعاملات
// يُرجى الرجوع إلى SPEC §3 Layer 1
// المسؤولية: استخراج publicId والبحث في قاعدة البيانات والتحقق من حالة السجل والجهاز
package gate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/model"
)

// GateService — خدمة طبقة البوابة
type GateService struct {
	payerRepo db.PayerRepo // مستودع سجلات الدافعين
}

// NewGateService — ينشئ نسخة خدمة البوابة
func NewGateService(payerRepo db.PayerRepo) *GateService {
	return &GateService{payerRepo: payerRepo}
}

// Process — يعالج طلب المعاملة في طبقة البوابة
// المنطق حسب SPEC §3 Layer 1:
//  1. استخراج publicId من paymentToken
//  2. البحث في DB عبر PayerRepo.FindByPublicId
//  3. إذا غير موجود → UNKNOWN_PAYER (404)
//  4. إذا status ≠ ACTIVE → ACCOUNT_SUSPENDED (403)
//  5. إذا deviceId مختلف → DEVICE_MISMATCH (403)
//  6. إرجاع GateResult
func (g *GateService) Process(ctx context.Context, req *model.TransactionRequest) (*model.GateResult, error) {
	publicId := req.PaymentToken.PublicId

	slog.Debug("البوابة: بدء المعالجة", "publicId", publicId)

	// 1. البحث عن سجل الدافع
	record, err := g.payerRepo.FindByPublicId(ctx, publicId)
	if err != nil {
		return nil, fmt.Errorf("البوابة: بحث الدافع %s: %w", publicId, err)
	}

	// 2. التحقق من وجود الدافع
	if record == nil {
		slog.Warn("البوابة: دافع غير مسجّل", "publicId", publicId)
		return nil, model.NewAppError(model.ErrUnknownPayer)
	}

	// 3. التحقق من حالة الحساب
	if record.Status != "ACTIVE" {
		slog.Warn("البوابة: حساب معلّق", "publicId", publicId, "status", record.Status)
		return nil, model.NewAppError(model.ErrAccountSuspended)
	}

	// 4. التحقق من مطابقة الجهاز
	if record.DeviceId != req.PaymentToken.DeviceId {
		slog.Warn("البوابة: جهاز غير مطابق",
			"publicId", publicId,
			"expected_device", record.DeviceId,
			"provided_device", req.PaymentToken.DeviceId,
		)
		return nil, model.NewAppError(model.ErrDeviceMismatch)
	}

	// 5. إرجاع نتيجة البوابة
	slog.Debug("البوابة: نجاح", "publicId", publicId, "walletId", record.WalletId)

	return &model.GateResult{
		PayerPublicId: record.PublicId,
		PayerWalletId: record.WalletId,
		SeedEncrypted: record.SeedEncrypted,
		SeedKeyID:     record.SeedKeyID,
		PayerCounter:  record.Counter,
		PayerLimit:    record.PayerLimit,
	}, nil
}
