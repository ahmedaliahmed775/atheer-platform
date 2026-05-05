// طبقة التنفيذ (EXECUTE) — الطبقة الثالثة من خط معالجة المعاملات
// يُرجى الرجوع إلى SPEC §3 Layer 3 — نمط Saga
// المسؤولية: خصم من الدافع، إيداع للتاجر، تعويض عند الفشل
package execute

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/model"
)

// getConnectionSource — يستخرج مصدر الاتصال من السياق
func getConnectionSource(ctx context.Context) string {
	if source, ok := ctx.Value(model.ConnectionSourceCtxKey{}).(string); ok && source != "" {
		return source
	}
	return model.SourceInternet
}

// ExecuteService — خدمة طبقة التنفيذ
type ExecuteService struct {
	adapter    model.WalletAdapter   // محوّل المحفظة
	payerRepo  db.PayerRepo          // مستودع الدافعين
	txRepo     db.TransactionRepo    // مستودع المعاملات
}

// NewExecuteService — ينشئ نسخة خدمة التنفيذ
func NewExecuteService(
	adapter model.WalletAdapter,
	payerRepo db.PayerRepo,
	txRepo db.TransactionRepo,
) *ExecuteService {
	return &ExecuteService{
		adapter:   adapter,
		payerRepo: payerRepo,
		txRepo:    txRepo,
	}
}

// Process — ينفّذ المعاملة بنمط Saga
// المنطق حسب SPEC §3 Layer 3:
//  1. resolveTransactionType → DIRECT أو CROSS_WALLET
//  2. CROSS_WALLET → خطأ CROSS_WALLET_NOT_SUPPORTED (الإصدار الأول)
//  3. Step 1: adapter.Debit(payerParams) → debitRef
//  4. Step 2: adapter.Credit(merchantParams) → creditRef
//  5. إذا فشل Credit → adapter.ReverseDebit(debitRef) ← تعويض
//  6. Step 3: payerRepo.UpdateCounter(publicId, newCounter)
//  7. Step 4: transactionRepo.Save(transaction)
//  8. إرجاع TransactionResponse
func (e *ExecuteService) Process(ctx context.Context, req *model.TransactionRequest, verify *model.VerifyResult) (*model.TransactionResponse, error) {
	startTime := time.Now()
	publicId := req.PaymentToken.PublicId

	slog.Debug("التنفيذ: بدء المعالجة",
		"publicId", publicId,
		"payerWalletId", verify.PayerWalletId,
		"merchantWalletId", verify.MerchantWalletId,
		"amount", verify.Amount,
	)

	// --- الخطوة 1: تحديد نوع المعاملة ---
	txType := resolveTransactionType(verify.PayerWalletId, verify.MerchantWalletId)
	if txType == CROSS_WALLET {
		slog.Warn("التنفيذ: معاملة بين محافظ مختلفة غير مدعومة",
			"payerWalletId", verify.PayerWalletId,
			"merchantWalletId", verify.MerchantWalletId,
		)
		return e.buildResponse(publicId, "FAILED", model.ErrCrossWalletNotSupported, verify.NewCounter, startTime), nil
	}

	// --- الخطوة 2: خصم من الدافع ---
	debitParams := model.DebitParams{
		WalletId:      verify.PayerWalletId,
		AccountRef:    publicId,
		Amount:        verify.Amount,
		Currency:      verify.Currency,
		IdempotencyKey: fmt.Sprintf("debit-%s-%d", publicId, verify.NewCounter),
	}

	debitResult, err := e.adapter.Debit(ctx, debitParams)
	if err != nil {
		slog.Error("التنفيذ: فشل الخصم",
			"publicId", publicId,
			"amount", verify.Amount,
			"error", err,
		)
		// حفظ المعاملة كفاشلة
		e.saveFailedTx(ctx, req, verify, model.ErrDebitFailed, "", "", startTime)
		return e.buildResponse(publicId, "FAILED", model.ErrDebitFailed, verify.NewCounter, startTime), nil
	}

	slog.Debug("التنفيذ: نجاح الخصم",
		"publicId", publicId,
		"debitRef", debitResult.DebitRef,
	)

	// --- الخطوة 3: إيداع للتاجر ---
	creditParams := model.CreditParams{
		WalletId:      verify.MerchantWalletId,
		AccountRef:    req.MerchantData.MerchantId,
		Amount:        verify.Amount,
		Currency:      verify.Currency,
		IdempotencyKey: fmt.Sprintf("credit-%s-%d", req.MerchantData.MerchantId, verify.NewCounter),
	}

	creditResult, err := e.adapter.Credit(ctx, creditParams)
	if err != nil {
		slog.Error("التنفيذ: فشل الإيداع — بدء التعويض",
			"publicId", publicId,
			"amount", verify.Amount,
			"debitRef", debitResult.DebitRef,
			"error", err,
		)

		// --- تعويض: عكس الخصم ---
		if reverseErr := e.compensate(ctx, publicId, debitResult.DebitRef); reverseErr != nil {
			slog.Error("التنفيذ: فشل التعويض أيضاً — تنبيه يدوي مطلوب",
				"publicId", publicId,
				"debitRef", debitResult.DebitRef,
				"reverseError", reverseErr,
			)
		}

		// حفظ المعاملة كفاشلة
		e.saveFailedTx(ctx, req, verify, model.ErrCreditFailed, debitResult.DebitRef, "", startTime)
		return e.buildResponse(publicId, "FAILED", model.ErrCreditFailed, verify.NewCounter, startTime), nil
	}

	slog.Debug("التنفيذ: نجاح الإيداع",
		"publicId", publicId,
		"creditRef", creditResult.CreditRef,
	)

	// --- الخطوة 4: تحديث العداد ---
	if err := e.payerRepo.UpdateCounter(ctx, publicId, verify.NewCounter); err != nil {
		slog.Error("التنفيذ: فشل تحديث العداد",
			"publicId", publicId,
			"newCounter", verify.NewCounter,
			"error", err,
		)
		// لا نُرجع فشل — المعاملة تمت لكن العداد لم يُحدَّث
		// سيتم إصلاحه لاحقاً عبر آلية المزامنة
	}

	// --- الخطوة 5: حفظ المعاملة ---
	durationMs := int(time.Since(startTime).Milliseconds())
	tx := &model.Transaction{
		TransactionId:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		PayerPublicId:    publicId,
		MerchantId:       req.MerchantData.MerchantId,
		PayerWalletId:    verify.PayerWalletId,
		MerchantWalletId: verify.MerchantWalletId,
		Amount:           verify.Amount,
		Currency:         verify.Currency,
		Counter:          verify.NewCounter,
		Status:           "SUCCESS",
		DurationMs:       durationMs,
		DebitRef:         debitResult.DebitRef,
		CreditRef:        creditResult.CreditRef,
		ConnectionSource: getConnectionSource(ctx),
	}

	if err := e.txRepo.Save(ctx, tx); err != nil {
		slog.Error("التنفيذ: فشل حفظ المعاملة",
			"transactionId", tx.TransactionId,
			"error", err,
		)
	}

	slog.Info("التنفيذ: نجاح المعاملة",
		"publicId", publicId,
		"transactionId", tx.TransactionId,
		"amount", verify.Amount,
		"duration_ms", durationMs,
	)

	return e.buildResponse(publicId, "SUCCESS", "", verify.NewCounter, startTime), nil
}

// compensate — يعكس عملية الخصم كتعويض عند فشل الإيداع
func (e *ExecuteService) compensate(ctx context.Context, publicId string, debitRef string) error {
	slog.Info("التنفيذ: تعويض — عكس الخصم", "publicId", publicId, "debitRef", debitRef)

	_, err := e.adapter.ReverseDebit(ctx, debitRef)
	if err != nil {
		return fmt.Errorf("التنفيذ: تعويض فاشل للخصم %s: %w", debitRef, err)
	}

	slog.Info("التنفيذ: نجاح التعويض", "publicId", publicId, "debitRef", debitRef)
	return nil
}

// saveFailedTx — يحفظ معاملة فاشلة في قاعدة البيانات
func (e *ExecuteService) saveFailedTx(ctx context.Context, req *model.TransactionRequest, verify *model.VerifyResult, errorCode string, debitRef string, creditRef string, startTime time.Time) {
	durationMs := int(time.Since(startTime).Milliseconds())
	tx := &model.Transaction{
		TransactionId:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		PayerPublicId:    req.PaymentToken.PublicId,
		MerchantId:       req.MerchantData.MerchantId,
		PayerWalletId:    verify.PayerWalletId,
		MerchantWalletId: verify.MerchantWalletId,
		Amount:           verify.Amount,
		Currency:         verify.Currency,
		Counter:          verify.NewCounter,
		Status:           "FAILED",
		ErrorCode:        errorCode,
		DurationMs:       durationMs,
		DebitRef:         debitRef,
		CreditRef:        creditRef,
		ConnectionSource: getConnectionSource(ctx),
	}

	if err := e.txRepo.Save(ctx, tx); err != nil {
		slog.Error("التنفيذ: فشل حفظ المعاملة الفاشلة",
			"transactionId", tx.TransactionId,
			"error", err,
		)
	}
}

// buildResponse — يبني استجابة المعاملة
func (e *ExecuteService) buildResponse(publicId, status, errorCode string, lastValidCounter int64, startTime time.Time) *model.TransactionResponse {
	resp := &model.TransactionResponse{
		TransactionId:    fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		Status:           status,
		ErrorCode:        errorCode,
		LastValidCounter: lastValidCounter,
		Timestamp:        time.Now().Unix(),
	}

	// إضافة رسالة الخطأ إن وُجدت
	if errorCode != "" {
		appErr := model.NewAppError(errorCode)
		resp.ErrorMessage = appErr.Message
	}

	return resp
}
