// معالج المعاملات — POST /api/v1/transaction
// يُرجى الرجوع إلى SPEC §3 و Task 07
// يسلسل الطبقات الثلاث: GATE → VERIFY → EXECUTE
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/atheer/switch/internal/execute"
	"github.com/atheer/switch/internal/gate"
	"github.com/atheer/switch/internal/model"
	"github.com/atheer/switch/internal/verify"
)

// TransactionHandler — معالج المعاملات
type TransactionHandler struct {
	gate    *gate.GateService
	verify  *verify.VerifyService
	execute *execute.ExecuteService
}

// NewTransactionHandler — ينشئ معالج معاملات جديد
func NewTransactionHandler(
	gate *gate.GateService,
	verify *verify.VerifyService,
	execute *execute.ExecuteService,
) *TransactionHandler {
	return &TransactionHandler{
		gate:    gate,
		verify:  verify,
		execute: execute,
	}
}

// Handle — يعالج طلب المعاملة عبر سلسلة GATE → VERIFY → EXECUTE
func (h *TransactionHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// 1. تحليل الطلب
	var req model.TransactionRequest
	if err := readJSON(r, &req); err != nil {
		slog.Warn("المعاملة: طلب غير صالح", "error", err)
		writeBadRequest(w, "جسم الطلب غير صالح")
		return
	}

	// 2. التحقق من الحقول المطلوبة
	if req.PaymentToken.PublicId == "" || req.PaymentToken.DeviceId == "" {
		writeBadRequest(w, "paymentToken.publicId و paymentToken.deviceId مطلوبان")
		return
	}
	if req.MerchantData.MerchantId == "" || req.MerchantData.MerchantWalletId == "" {
		writeBadRequest(w, "merchantData.merchantId و merchantData.merchantWalletId مطلوبان")
		return
	}
	if req.MerchantData.Amount <= 0 {
		writeBadRequest(w, "merchantData.amount يجب أن يكون أكبر من صفر")
		return
	}

	slog.Debug("المعاملة: بدء المعالجة",
		"publicId", req.PaymentToken.PublicId,
		"merchantId", req.MerchantData.MerchantId,
		"amount", req.MerchantData.Amount,
	)

	// 3. طبقة البوابة (GATE)
	gateResult, err := h.gate.Process(ctx, &req)
	if err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			slog.Warn("المعاملة: فشل في البوابة",
				"errorCode", appErr.Code,
				"publicId", req.PaymentToken.PublicId,
			)
			writeError(w, appErr)
			return
		}
		slog.Error("المعاملة: خطأ داخلي في البوابة", "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}

	// 4. طبقة التحقق (VERIFY)
	verifyResult, err := h.verify.Process(ctx, &req, gateResult)
	if err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			slog.Warn("المعاملة: فشل في التحقق",
				"errorCode", appErr.Code,
				"publicId", req.PaymentToken.PublicId,
			)
			writeError(w, appErr)
			return
		}
		slog.Error("المعاملة: خطأ داخلي في التحقق", "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}

	// 5. طبقة التنفيذ (EXECUTE)
	response, err := h.execute.Process(ctx, &req, verifyResult)
	if err != nil {
		slog.Error("المعاملة: خطأ داخلي في التنفيذ", "error", err)
		http.Error(w, "خطأ داخلي", http.StatusInternalServerError)
		return
	}

	// 6. إرجاع الاستجابة
	duration := time.Since(startTime)
	slog.Info("المعاملة: اكتملت",
		"publicId", req.PaymentToken.PublicId,
		"status", response.Status,
		"duration_ms", duration.Milliseconds(),
	)

	statusCode := http.StatusOK
	if response.Status == "FAILED" {
		statusCode = http.StatusOK // الخطأ منطقي وليس HTTP
	}
	writeJSON(w, statusCode, response)
}
