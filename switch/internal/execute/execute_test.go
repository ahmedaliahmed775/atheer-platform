// اختبارات طبقة التنفيذ (EXECUTE) — يُرجى الرجوع إلى SPEC §3 Layer 3
package execute

import (
	"context"
	"fmt"
	"testing"

	"github.com/atheer/switch/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- واجهات وهمية للاختبار ---

// mockWalletAdapter — محوّل محفظة وهمي
type mockWalletAdapter struct {
	debitResult    *model.DebitResult
	debitErr       error
	creditResult   *model.CreditResult
	creditErr      error
	reverseResult  *model.ReverseResult
	reverseErr     error
	reverseCalled  bool
	verifyResult   bool
	verifyErr      error
}

func (m *mockWalletAdapter) VerifyAccessToken(_ context.Context, _, _ string) (bool, error) {
	return m.verifyResult, m.verifyErr
}

func (m *mockWalletAdapter) Debit(_ context.Context, _ model.DebitParams) (*model.DebitResult, error) {
	return m.debitResult, m.debitErr
}

func (m *mockWalletAdapter) Credit(_ context.Context, _ model.CreditParams) (*model.CreditResult, error) {
	return m.creditResult, m.creditErr
}

func (m *mockWalletAdapter) ReverseDebit(_ context.Context, _ string) (*model.ReverseResult, error) {
	m.reverseCalled = true
	return m.reverseResult, m.reverseErr
}

func (m *mockWalletAdapter) QueryTransaction(_ context.Context, _ string) (*model.TxStatus, error) {
	return nil, nil
}

// mockPayerRepoExecute — مستودع دافعين وهمي للتنفيذ
type mockPayerRepoExecute struct {
	counterUpdated bool
	err            error
}

func (m *mockPayerRepoExecute) FindByPublicId(_ context.Context, _ string) (*model.SwitchRecord, error) {
	return nil, nil
}

func (m *mockPayerRepoExecute) Create(_ context.Context, _ *model.SwitchRecord) error {
	return nil
}

func (m *mockPayerRepoExecute) UpdateCounter(_ context.Context, _ string, _ int64) error {
	m.counterUpdated = true
	return m.err
}

func (m *mockPayerRepoExecute) UpdateStatus(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockPayerRepoExecute) UpdatePayerLimit(_ context.Context, _ string, _ int64) error {
	return nil
}

func (m *mockPayerRepoExecute) Delete(_ context.Context, _ string) error {
	return nil
}

// mockTxRepoExecute — مستودع معاملات وهمي للتنفيذ
type mockTxRepoExecute struct {
	saved bool
	err   error
}

func (m *mockTxRepoExecute) Save(_ context.Context, _ *model.Transaction) error {
	m.saved = true
	return m.err
}

func (m *mockTxRepoExecute) FindByID(_ context.Context, _ string) (*model.Transaction, error) {
	return nil, nil
}

func (m *mockTxRepoExecute) List(_ context.Context, _ model.TransactionFilters, _, _ int) ([]model.Transaction, int, error) {
	return nil, 0, nil
}

func (m *mockTxRepoExecute) GetDailyTotal(_ context.Context, _ string, _ string) (int64, error) {
	return 0, nil
}

func (m *mockTxRepoExecute) GetMonthlyTotal(_ context.Context, _ string, _ string) (int64, error) {
	return 0, nil
}

// --- بيانات الاختبار ---

func makeValidExecuteRequest() *model.TransactionRequest {
	return &model.TransactionRequest{
		PaymentToken: model.PaymentToken{
			PublicId:  "usr_abc123",
			DeviceId:  "dev_456",
			Counter:   43,
			Timestamp: 1714340400,
			HMAC:      "base64-hmac",
		},
		MerchantData: model.MerchantData{
			MerchantId:       "770123456",
			MerchantWalletId: "jawali",
			Amount:           2500,
			Currency:         "YER",
			AccessToken:      "valid-token",
		},
		Timestamp: 1714340400,
	}
}

func makeValidVerifyResult() *model.VerifyResult {
	return &model.VerifyResult{
		PayerWalletId:    "jawali",
		MerchantWalletId: "jawali", // نفس المحفظة = DIRECT
		Amount:           2500,
		Currency:         "YER",
		NewCounter:       43,
	}
}

// TestExecute_DirectSuccess — معاملة مباشرة ناجحة: خصم + إيداع
func TestExecute_DirectSuccess(t *testing.T) {
	adapter := &mockWalletAdapter{
		debitResult:  &model.DebitResult{DebitRef: "debit-ref-001", Status: "SUCCESS"},
		creditResult: &model.CreditResult{CreditRef: "credit-ref-001", Status: "SUCCESS"},
	}
	payerRepo := &mockPayerRepoExecute{}
	txRepo := &mockTxRepoExecute{}

	svc := NewExecuteService(adapter, payerRepo, txRepo)

	resp, err := svc.Process(context.Background(), makeValidExecuteRequest(), makeValidVerifyResult())
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Empty(t, resp.ErrorCode)
	assert.Equal(t, int64(43), resp.LastValidCounter)
	assert.True(t, payerRepo.counterUpdated, "يجب تحديث العداد")
	assert.True(t, txRepo.saved, "يجب حفظ المعاملة")
	assert.False(t, adapter.reverseCalled, "لا يجب استدعاء التعويض")
}

// TestExecute_DebitFailed — فشل الخصم → لا إيداع، لا تعويض
func TestExecute_DebitFailed(t *testing.T) {
	adapter := &mockWalletAdapter{
		debitErr: fmt.Errorf("فشل الاتصال بالمحفظة"),
	}
	payerRepo := &mockPayerRepoExecute{}
	txRepo := &mockTxRepoExecute{}

	svc := NewExecuteService(adapter, payerRepo, txRepo)

	resp, err := svc.Process(context.Background(), makeValidExecuteRequest(), makeValidVerifyResult())
	require.NoError(t, err) // لا خطأ من الدالة — الخطأ في الاستجابة
	require.NotNil(t, resp)

	assert.Equal(t, "FAILED", resp.Status)
	assert.Equal(t, model.ErrDebitFailed, resp.ErrorCode)
	assert.False(t, adapter.reverseCalled, "لا يجب استدعاء التعويض عند فشل الخصم")
	assert.False(t, payerRepo.counterUpdated, "لا يجب تحديث العداد عند الفشل")
	assert.True(t, txRepo.saved, "يجب حفظ المعاملة الفاشلة")
}

// TestExecute_CreditFailed_Compensate — فشل الإيداع → تعويض بعكس الخصم
func TestExecute_CreditFailed_Compensate(t *testing.T) {
	adapter := &mockWalletAdapter{
		debitResult:  &model.DebitResult{DebitRef: "debit-ref-002", Status: "SUCCESS"},
		creditErr:    fmt.Errorf("فشل الإيداع في المحفظة"),
		reverseResult: &model.ReverseResult{ReverseRef: "reverse-ref-002", Status: "SUCCESS"},
	}
	payerRepo := &mockPayerRepoExecute{}
	txRepo := &mockTxRepoExecute{}

	svc := NewExecuteService(adapter, payerRepo, txRepo)

	resp, err := svc.Process(context.Background(), makeValidExecuteRequest(), makeValidVerifyResult())
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "FAILED", resp.Status)
	assert.Equal(t, model.ErrCreditFailed, resp.ErrorCode)
	assert.True(t, adapter.reverseCalled, "يجب استدعاء التعويض عند فشل الإيداع")
	assert.False(t, payerRepo.counterUpdated, "لا يجب تحديث العداد عند الفشل")
	assert.True(t, txRepo.saved, "يجب حفظ المعاملة الفاشلة")
}

// TestExecute_CrossWallet_Rejected — معاملة بين محافظ مختلفة مرفوضة
func TestExecute_CrossWallet_Rejected(t *testing.T) {
	adapter := &mockWalletAdapter{
		debitResult:  &model.DebitResult{DebitRef: "debit-ref-003", Status: "SUCCESS"},
		creditResult: &model.CreditResult{CreditRef: "credit-ref-003", Status: "SUCCESS"},
	}
	payerRepo := &mockPayerRepoExecute{}
	txRepo := &mockTxRepoExecute{}

	svc := NewExecuteService(adapter, payerRepo, txRepo)

	// تعديل نتيجة التحقق لتكون بين محافظ مختلفة
	verify := makeValidVerifyResult()
	verify.MerchantWalletId = "flousk" // محفظة مختلفة عن الدافع

	resp, err := svc.Process(context.Background(), makeValidExecuteRequest(), verify)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "FAILED", resp.Status)
	assert.Equal(t, model.ErrCrossWalletNotSupported, resp.ErrorCode)
	assert.False(t, adapter.reverseCalled, "لا يجب استدعاء أي عملية محفظة")
	assert.False(t, payerRepo.counterUpdated, "لا يجب تحديث العداد")
}
