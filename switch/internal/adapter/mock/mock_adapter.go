// محوّل محفظة وهمي للاختبارات — يستخدم حقول دوال لسهولة التخصيص
// يُرجى الرجوع إلى skills/adapter.md — Mock Adapter
package mock

import (
	"context"

	"github.com/atheer/switch/internal/model"
)

// MockAdapter — محوّل محفظة وهمي للاختبارات
// كل دالة قابلة للتخصيص عبر حقول الدوال — القيم الافتراضية تُرجع نجاح
type MockAdapter struct {
	VerifyAccessTokenFunc func(ctx context.Context, walletId, accessToken string) (bool, error)
	DebitFunc             func(ctx context.Context, params model.DebitParams) (*model.DebitResult, error)
	CreditFunc            func(ctx context.Context, params model.CreditParams) (*model.CreditResult, error)
	ReverseDebitFunc      func(ctx context.Context, debitRef string) (*model.ReverseResult, error)
	QueryTransactionFunc  func(ctx context.Context, txRef string) (*model.TxStatus, error)
}

// VerifyAccessToken — يتحقق من رمز وصول التاجر
func (m *MockAdapter) VerifyAccessToken(ctx context.Context, walletId, accessToken string) (bool, error) {
	if m.VerifyAccessTokenFunc != nil {
		return m.VerifyAccessTokenFunc(ctx, walletId, accessToken)
	}
	return true, nil // افتراضي: نجاح
}

// Debit — يخصم مبلغاً من حساب الدافع
func (m *MockAdapter) Debit(ctx context.Context, params model.DebitParams) (*model.DebitResult, error) {
	if m.DebitFunc != nil {
		return m.DebitFunc(ctx, params)
	}
	return &model.DebitResult{
		DebitRef: "mock-debit-ref",
		Status:   "SUCCESS",
	}, nil
}

// Credit — يودع مبلغاً في حساب التاجر
func (m *MockAdapter) Credit(ctx context.Context, params model.CreditParams) (*model.CreditResult, error) {
	if m.CreditFunc != nil {
		return m.CreditFunc(ctx, params)
	}
	return &model.CreditResult{
		CreditRef: "mock-credit-ref",
		Status:    "SUCCESS",
	}, nil
}

// ReverseDebit — يعكس عملية خصم سابقة
func (m *MockAdapter) ReverseDebit(ctx context.Context, debitRef string) (*model.ReverseResult, error) {
	if m.ReverseDebitFunc != nil {
		return m.ReverseDebitFunc(ctx, debitRef)
	}
	return &model.ReverseResult{
		ReverseRef: "mock-reverse-ref",
		Status:     "SUCCESS",
	}, nil
}

// QueryTransaction — يستعلم عن حالة معاملة
func (m *MockAdapter) QueryTransaction(ctx context.Context, txRef string) (*model.TxStatus, error) {
	if m.QueryTransactionFunc != nil {
		return m.QueryTransactionFunc(ctx, txRef)
	}
	return &model.TxStatus{
		Ref:    txRef,
		Status: "SUCCESS",
	}, nil
}
