// طبقة التحقق (VERIFY) — الطبقة الثانية من خط معالجة المعاملات
// يُرجى الرجوع إلى SPEC §3 Layer 2
// المسؤولية: التحقق من التاجر، الطابع الزمني، العداد، HMAC، والحدود
package verify

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/atheer/switch/internal/crypto"
	"github.com/atheer/switch/internal/model"
)

// MerchantVerifier — واجهة التحقق من رمز وصول التاجر
// سيتحقق عبر محوّل المحفظة المناسب
type MerchantVerifier interface {
	VerifyAccessToken(ctx context.Context, walletId, accessToken string) (bool, error)
}

// VerifyService — خدمة طبقة التحقق
type VerifyService struct {
	kms             crypto.KMS          // نظام إدارة المفاتيح لفك تشفير البذرة
	merchantVerifier MerchantVerifier   // محقّق رمز وصول التاجر
	limitsChecker   *LimitsChecker      // فاحص الحدود اليومية/الشهرية
	timestampTol    int64               // تسامح الطابع الزمني بالثواني (60)
	lookAheadWindow int64               // نافذة العداد المسموحة (10)
}

// NewVerifyService — ينشئ نسخة خدمة التحقق
func NewVerifyService(
	kms crypto.KMS,
	merchantVerifier MerchantVerifier,
	limitsChecker *LimitsChecker,
	timestampTolerance int64,
	lookAheadWindow int64,
) *VerifyService {
	return &VerifyService{
		kms:             kms,
		merchantVerifier: merchantVerifier,
		limitsChecker:   limitsChecker,
		timestampTol:    timestampTolerance,
		lookAheadWindow: lookAheadWindow,
	}
}

// Process — يعالج طلب المعاملة في طبقة التحقق
// المنطق حسب SPEC §3 Layer 2:
//  1. التحقق من accessToken التاجر → MERCHANT_UNAUTHORIZED
//  2. فحص timestamp → TIMESTAMP_EXPIRED
//  3. فحص counter replay → COUNTER_REPLAY
//  4. فحص counter window → COUNTER_OUT_OF_WINDOW
//  5. فك تشفير البذرة → اشتقاق LUK → فحص HMAC → HMAC_MISMATCH
//  6. فحص amount ≤ payerLimit → PAYER_LIMIT_EXCEEDED
//  7. فحص حدود يومي/شهري → LIMIT_EXCEEDED
func (v *VerifyService) Process(ctx context.Context, req *model.TransactionRequest, gate *model.GateResult) (*model.VerifyResult, error) {
	publicId := req.PaymentToken.PublicId
	slog.Debug("التحقق: بدء المعالجة", "publicId", publicId)

	// --- الخطوة 1: التحقق من رمز وصول التاجر ---
	valid, err := v.merchantVerifier.VerifyAccessToken(ctx, req.MerchantData.MerchantWalletId, req.MerchantData.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("التحقق: مصادقة التاجر: %w", err)
	}
	if !valid {
		slog.Warn("التحقق: رمز وصول التاجر غير صالح",
			"merchantWalletId", req.MerchantData.MerchantWalletId,
		)
		return nil, model.NewAppError(model.ErrMerchantUnauthorized)
	}

	// --- الخطوة 2: فحص الطابع الزمني ---
	now := time.Now().Unix()
	diff := req.PaymentToken.Timestamp - now
	if diff < 0 {
		diff = -diff // القيمة المطلقة
	}
	if diff > v.timestampTol {
		slog.Warn("التحقق: طابع زمني منتهي",
			"publicId", publicId,
			"token_ts", req.PaymentToken.Timestamp,
			"now", now,
			"diff", diff,
		)
		return nil, model.NewAppError(model.ErrTimestampExpired)
	}

	// --- الخطوة 3: فحص إعادة تشغيل العداد ---
	if req.PaymentToken.Counter <= gate.PayerCounter {
		slog.Warn("التحقق: عداد إعادة تشغيل",
			"publicId", publicId,
			"token_counter", req.PaymentToken.Counter,
			"stored_counter", gate.PayerCounter,
		)
		return nil, model.NewAppError(model.ErrCounterReplay)
	}

	// --- الخطوة 4: فحص نافذة العداد ---
	if req.PaymentToken.Counter > gate.PayerCounter+v.lookAheadWindow {
		slog.Warn("التحقق: عداد خارج النافذة",
			"publicId", publicId,
			"token_counter", req.PaymentToken.Counter,
			"stored_counter", gate.PayerCounter,
			"window", v.lookAheadWindow,
		)
		return nil, model.NewAppError(model.ErrCounterOutOfWindow)
	}

	// --- الخطوة 5: فك تشفير البذرة → اشتقاق LUK → فحص HMAC ---
	seed, err := v.kms.Decrypt(ctx, gate.SeedKeyID, gate.SeedEncrypted)
	if err != nil {
		return nil, fmt.Errorf("التحقق: فك تشفير البذرة: %w", err)
	}
	defer crypto.Zeroize(seed)

	luk, err := crypto.DeriveLUK(seed)
	if err != nil {
		return nil, fmt.Errorf("التحقق: اشتقاق LUK: %w", err)
	}
	defer crypto.Zeroize(luk)

	// فك تشفير HMAC من base64
	providedHMAC, err := base64.StdEncoding.DecodeString(req.PaymentToken.HMAC)
	if err != nil {
		slog.Warn("التحقق: HMAC بصيغة غير صالحة", "publicId", publicId)
		return nil, model.NewAppError(model.ErrHMACMismatch)
	}

	hmacValid, err := crypto.VerifyHMAC(
		luk, publicId, req.PaymentToken.DeviceId,
		req.PaymentToken.Counter, req.PaymentToken.Timestamp,
		providedHMAC,
	)
	if err != nil {
		return nil, fmt.Errorf("التحقق: حساب HMAC: %w", err)
	}
	if !hmacValid {
		slog.Warn("التحقق: HMAC غير مطابق", "publicId", publicId)
		return nil, model.NewAppError(model.ErrHMACMismatch)
	}

	// --- الخطوة 6: فحص حد الدافع ---
	if req.MerchantData.Amount > gate.PayerLimit {
		slog.Warn("التحقق: المبلغ يتجاوز حد الدافع",
			"publicId", publicId,
			"amount", req.MerchantData.Amount,
			"payerLimit", gate.PayerLimit,
		)
		return nil, model.NewAppError(model.ErrPayerLimitExceeded)
	}

	// --- الخطوة 7: فحص الحدود اليومية/الشهرية ---
	if err := v.limitsChecker.CheckLimits(ctx, publicId, req.MerchantData.Amount); err != nil {
		return nil, err
	}

	// --- نجاح: إرجاع نتيجة التحقق ---
	slog.Debug("التحقق: نجاح", "publicId", publicId)

	return &model.VerifyResult{
		PayerWalletId:    gate.PayerWalletId,
		MerchantWalletId: req.MerchantData.MerchantWalletId,
		Amount:           req.MerchantData.Amount,
		Currency:         req.MerchantData.Currency,
		NewCounter:       req.PaymentToken.Counter,
	}, nil
}
