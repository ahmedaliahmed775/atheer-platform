// اختبارات طبقة البوابة (GATE) — يُرجى الرجوع إلى SPEC §3 Layer 1
package gate

import (
	"context"
	"testing"

	"github.com/atheer/switch/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPayerRepo — مستودع دافعين وهمي للاختبار
type mockPayerRepo struct {
	record *model.SwitchRecord // السجل المُعاد
	err    error               // الخطأ المُعاد
}

func (m *mockPayerRepo) FindByPublicId(_ context.Context, _ string) (*model.SwitchRecord, error) {
	return m.record, m.err
}

func (m *mockPayerRepo) Create(_ context.Context, _ *model.SwitchRecord) error {
	return nil
}

func (m *mockPayerRepo) UpdateCounter(_ context.Context, _ string, _ int64) error {
	return nil
}

func (m *mockPayerRepo) UpdateStatus(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockPayerRepo) UpdatePayerLimit(_ context.Context, _ string, _ int64) error {
	return nil
}

func (m *mockPayerRepo) Delete(_ context.Context, _ string) error {
	return nil
}

// سجل دافع صالح للاختبار
func validRecord() *model.SwitchRecord {
	return &model.SwitchRecord{
		ID:            1,
		PublicId:      "usr_abc123",
		WalletId:      "jawali",
		DeviceId:      "dev_456",
		SeedEncrypted: []byte("encrypted-seed-data"),
		SeedKeyID:     "local-v1",
		Counter:       42,
		PayerLimit:    5000,
		Status:        "ACTIVE",
		UserType:      "P",
	}
}

// طلب معاملة صالح للاختبار
func validRequest() *model.TransactionRequest {
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
			AccessToken:      "xxx",
		},
		Timestamp: 1714340400,
	}
}

// TestGate_Success — دافع مسجّل ونشط وجهاز مطابق → نجاح
func TestGate_Success(t *testing.T) {
	repo := &mockPayerRepo{record: validRecord()}
	svc := NewGateService(repo)

	result, err := svc.Process(context.Background(), validRequest())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "usr_abc123", result.PayerPublicId)
	assert.Equal(t, "jawali", result.PayerWalletId)
	assert.Equal(t, []byte("encrypted-seed-data"), result.SeedEncrypted)
	assert.Equal(t, "local-v1", result.SeedKeyID)
	assert.Equal(t, int64(42), result.PayerCounter)
	assert.Equal(t, int64(5000), result.PayerLimit)
}

// TestGate_UnknownPayer — دافع غير مسجّل → UNKNOWN_PAYER
func TestGate_UnknownPayer(t *testing.T) {
	repo := &mockPayerRepo{record: nil} // لا سجل
	svc := NewGateService(repo)

	result, err := svc.Process(context.Background(), validRequest())
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrUnknownPayer, appErr.Code)
	assert.Equal(t, 404, appErr.HTTPStatus)
}

// TestGate_Suspended — حساب معلّق → ACCOUNT_SUSPENDED
func TestGate_Suspended(t *testing.T) {
	record := validRecord()
	record.Status = "SUSPENDED"
	repo := &mockPayerRepo{record: record}
	svc := NewGateService(repo)

	result, err := svc.Process(context.Background(), validRequest())
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrAccountSuspended, appErr.Code)
	assert.Equal(t, 403, appErr.HTTPStatus)
}

// TestGate_DeviceMismatch — جهاز غير مطابق → DEVICE_MISMATCH
func TestGate_DeviceMismatch(t *testing.T) {
	record := validRecord()
	record.DeviceId = "dev_OTHER" // جهاز مختلف
	repo := &mockPayerRepo{record: record}
	svc := NewGateService(repo)

	req := validRequest()
	// req.PaymentToken.DeviceId = "dev_456" لكن السجل فيه "dev_OTHER"

	result, err := svc.Process(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrDeviceMismatch, appErr.Code)
	assert.Equal(t, 403, appErr.HTTPStatus)
}
