// محوّل محفظة جوالي — تنفيذ واجهة WalletAdapter
// يُرجى الرجوع إلى Task 08 — Jawali API Mapping
// العمليات: VerifyAccessToken, Debit, Credit, ReverseDebit, QueryTransaction
package jawali

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atheer/switch/internal/model"
)

// JawaliAdapter — محوّل محفظة جوالي يُنفّذ واجهة WalletAdapter
// كل استدعاء يمرّ عبر قاطع الدائرة (CircuitBreaker)
type JawaliAdapter struct {
	client *Client // عميل HTTP مع إعادة المحاولة وقاطع الدائرة
}

// NewJawaliAdapter — ينشئ محوّل محفظة جوالي جديد
// يأخذ إعدادات المحفظة من wallet_configs
func NewJawaliAdapter(cfg ClientConfig) *JawaliAdapter {
	return &JawaliAdapter{
		client: NewClient(cfg),
	}
}

// NewJawaliAdapterWithClient — ينشئ محوّل جوالي مع عميل مخصّص (للاختبارات)
func NewJawaliAdapterWithClient(client *Client) *JawaliAdapter {
	return &JawaliAdapter{
		client: client,
	}
}

// VerifyAccessToken — يتحقق من صحة رمز وصول التاجر
// المنهج: POST /auth/verify
func (j *JawaliAdapter) VerifyAccessToken(ctx context.Context, walletId, accessToken string) (bool, error) {
	req := JawaliAuthVerifyRequest{
		AccessToken: accessToken,
		WalletId:    walletId,
	}

	resp, err := postAndDecode[JawaliAuthVerifyResponse](j.client, ctx, "/auth/verify", req)
	if err != nil {
		slog.Error("جوالي: فشل التحقق من الرمز",
			"walletId", walletId,
			"error", err,
		)
		return false, fmt.Errorf("جوالي: التحقق من الرمز: %w", err)
	}

	if !resp.IsSuccess() {
		slog.Warn("جوالي: رمز غير صالح",
			"walletId", walletId,
			"responseCode", resp.ResponseCode,
			"responseMessage", resp.ResponseMessage,
		)
		return false, nil // ليس خطأ شبكة — الرمز غير صالح
	}

	slog.Debug("جوالي: التحقق من الرمز ناجح", "walletId", walletId)
	return resp.Valid, nil
}

// Debit — يخصم مبلغاً من حساب الدافع
// المنهج: POST /cashout (ECOMMCASHOUT)
func (j *JawaliAdapter) Debit(ctx context.Context, params model.DebitParams) (*model.DebitResult, error) {
	req := JawaliCashoutRequest{
		MerchantId: params.AccountRef,    // مرجع حساب التاجر
		PayerPhone: params.WalletId,      // رقم هاتف الدافع (أو معرّف المحفظة)
		Amount:     params.Amount,
		Currency:   params.Currency,
		Reference:  params.IdempotencyKey,
	}

	resp, err := postAndDecode[JawaliCashoutResponse](j.client, ctx, "/cashout", req)
	if err != nil {
		slog.Error("جوالي: فشل الخصم",
			"payerPhone", req.PayerPhone,
			"amount", params.Amount,
			"error", err,
		)
		return nil, fmt.Errorf("جوالي: الخصم: %w", err)
	}

	if !resp.IsSuccess() {
		slog.Warn("جوالي: الخصم مرفوض",
			"responseCode", resp.ResponseCode,
			"responseMessage", resp.ResponseMessage,
			"payerPhone", req.PayerPhone,
			"amount", params.Amount,
		)
		return &model.DebitResult{
			DebitRef: resp.TransactionRef,
			Status:   "FAILED",
		}, nil
	}

	slog.Info("جوالي: الخصم ناجح",
		"debitRef", resp.TransactionRef,
		"payerPhone", req.PayerPhone,
		"amount", params.Amount,
	)
	return &model.DebitResult{
		DebitRef: resp.TransactionRef,
		Status:   "SUCCESS",
	}, nil
}

// Credit — يودع مبلغاً في حساب التاجر
// المنهج: POST /cashin
func (j *JawaliAdapter) Credit(ctx context.Context, params model.CreditParams) (*model.CreditResult, error) {
	req := JawaliCashinRequest{
		MerchantId: params.AccountRef,
		AccountRef: params.AccountRef,
		Amount:     params.Amount,
		Currency:   params.Currency,
		Reference:  params.IdempotencyKey,
	}

	resp, err := postAndDecode[JawaliCashinResponse](j.client, ctx, "/cashin", req)
	if err != nil {
		slog.Error("جوالي: فشل الإيداع",
			"merchantId", req.MerchantId,
			"amount", params.Amount,
			"error", err,
		)
		return nil, fmt.Errorf("جوالي: الإيداع: %w", err)
	}

	if !resp.IsSuccess() {
		slog.Warn("جوالي: الإيداع مرفوض",
			"responseCode", resp.ResponseCode,
			"responseMessage", resp.ResponseMessage,
			"merchantId", req.MerchantId,
			"amount", params.Amount,
		)
		return &model.CreditResult{
			CreditRef: resp.TransactionRef,
			Status:    "FAILED",
		}, nil
	}

	slog.Info("جوالي: الإيداع ناجح",
		"creditRef", resp.TransactionRef,
		"merchantId", req.MerchantId,
		"amount", params.Amount,
	)
	return &model.CreditResult{
		CreditRef: resp.TransactionRef,
		Status:    "SUCCESS",
	}, nil
}

// ReverseDebit — يعكس عملية خصم سابقة (تعويض في Saga)
// المنهج: POST /cashout/reverse
func (j *JawaliAdapter) ReverseDebit(ctx context.Context, debitRef string) (*model.ReverseResult, error) {
	req := JawaliReverseRequest{
		OriginalReference: debitRef,
		Reason:            "Atheer Saga compensation",
	}

	resp, err := postAndDecode[JawaliReverseResponse](j.client, ctx, "/cashout/reverse", req)
	if err != nil {
		slog.Error("جوالي: فشل عكس الخصم",
			"debitRef", debitRef,
			"error", err,
		)
		return nil, fmt.Errorf("جوالي: عكس الخصم: %w", err)
	}

	if !resp.IsSuccess() {
		slog.Warn("جوالي: العكس مرفوض",
			"responseCode", resp.ResponseCode,
			"responseMessage", resp.ResponseMessage,
			"debitRef", debitRef,
		)
		return &model.ReverseResult{
			ReverseRef: resp.ReverseRef,
			Status:     "FAILED",
		}, nil
	}

	slog.Info("جوالي: عكس الخصم ناجح",
		"reverseRef", resp.ReverseRef,
		"originalDebitRef", debitRef,
	)
	return &model.ReverseResult{
		ReverseRef: resp.ReverseRef,
		Status:     "SUCCESS",
	}, nil
}

// QueryTransaction — يستعلم عن حالة معاملة في محفظة جوالي
// المنهج: POST /inquiry (ECOMMERCEINQUIRY)
func (j *JawaliAdapter) QueryTransaction(ctx context.Context, txRef string) (*model.TxStatus, error) {
	req := JawaliInquiryRequest{
		Reference: txRef,
	}

	resp, err := postAndDecode[JawaliInquiryResponse](j.client, ctx, "/inquiry", req)
	if err != nil {
		slog.Error("جوالي: فشل الاستعلام",
			"txRef", txRef,
			"error", err,
		)
		return nil, fmt.Errorf("جوالي: الاستعلام: %w", err)
	}

	if !resp.IsSuccess() {
		slog.Warn("جوالي: الاستعلام مرفوض",
			"responseCode", resp.ResponseCode,
			"responseMessage", resp.ResponseMessage,
			"txRef", txRef,
		)
		return &model.TxStatus{
			Ref:    txRef,
			Status: "UNKNOWN",
		}, nil
	}

	slog.Debug("جوالي: الاستعلام ناجح",
		"txRef", txRef,
		"status", resp.Status,
	)
	return &model.TxStatus{
		Ref:    resp.TransactionRef,
		Status: resp.Status,
	}, nil
}
