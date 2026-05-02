// اختبارات طبقة التحقق (VERIFY) — يُرجى الرجوع إلى SPEC §3 Layer 2
package verify

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/atheer/switch/internal/crypto"
	"github.com/atheer/switch/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- واجهات وهمية للاختبار ---

// mockMerchantVerifier — محقّق تاجر وهمي
type mockMerchantVerifier struct {
	valid bool
	err   error
}

func (m *mockMerchantVerifier) VerifyAccessToken(_ context.Context, _, _ string) (bool, error) {
	return m.valid, m.err
}

// mockKMS — نظام إدارة مفاتيح وهمي
type mockKMS struct {
	decrypted []byte
	err       error
}

func (m *mockKMS) Encrypt(_ context.Context, _ []byte) ([]byte, string, error) {
	return nil, "", nil
}

func (m *mockKMS) Decrypt(_ context.Context, _ string, _ []byte) ([]byte, error) {
	return m.decrypted, m.err
}

// mockTxRepo — مستودع معاملات وهمي
type mockTxRepo struct {
	dailyTotal   int64
	monthlyTotal int64
	err          error
}

func (m *mockTxRepo) Save(_ context.Context, _ *model.Transaction) error {
	return nil
}

func (m *mockTxRepo) FindByID(_ context.Context, _ string) (*model.Transaction, error) {
	return nil, nil
}

func (m *mockTxRepo) List(_ context.Context, _ model.TransactionFilters, _, _ int) ([]model.Transaction, int, error) {
	return nil, 0, nil
}

func (m *mockTxRepo) GetDailyTotal(_ context.Context, _ string, _ string) (int64, error) {
	return m.dailyTotal, m.err
}

func (m *mockTxRepo) GetMonthlyTotal(_ context.Context, _ string, _ string) (int64, error) {
	return m.monthlyTotal, m.err
}

// --- بيانات الاختبار ---

// مفتاح رئيسي لـ KMS المحلي
var testMasterKey = make([]byte, 32)

func init() {
	// إنشاء مفتاح رئيسي ثابت للاختبار
	for i := range testMasterKey {
		testMasterKey[i] = byte(i)
	}
}

// بذرة اختبار ثابتة (32 بايت)
var testSeed = []byte("test-seed-32-bytes-for-verify!!!")

// إنشاء طلب معاملة صالح مع HMAC صحيح
func makeValidRequest() (*model.TransactionRequest, *model.GateResult, *crypto.LocalKMS) {
	kms, _ := crypto.NewLocalKMS(testMasterKey)

	// تشفير البذرة
	seedEncrypted, seedKeyID, _ := kms.Encrypt(context.Background(), testSeed)

	// اشتقاق LUK من البذرة
	luk, _ := crypto.DeriveLUK(testSeed)

	// حساب HMAC الصحيح
	publicId := "usr_abc123"
	deviceId := "dev_456"
	counter := int64(43)
	timestamp := time.Now().Unix()

	data := fmt.Sprintf("%s|%s|%d|%d", publicId, deviceId, counter, timestamp)
	mac := hmac.New(sha256.New, luk)
	mac.Write([]byte(data))
	hmacBytes := mac.Sum(nil)
	crypto.Zeroize(luk)

	req := &model.TransactionRequest{
		PaymentToken: model.PaymentToken{
			PublicId:  publicId,
			DeviceId:  deviceId,
			Counter:   counter,
			Timestamp: timestamp,
			HMAC:      base64.StdEncoding.EncodeToString(hmacBytes),
		},
		MerchantData: model.MerchantData{
			MerchantId:       "770123456",
			MerchantWalletId: "jawali",
			Amount:           2500,
			Currency:         "YER",
			AccessToken:      "valid-token",
		},
		Timestamp: timestamp,
	}

	gate := &model.GateResult{
		PayerPublicId: publicId,
		PayerWalletId: "jawali",
		SeedEncrypted: seedEncrypted,
		SeedKeyID:     seedKeyID,
		PayerCounter:  42,
		PayerLimit:    5000,
	}

	return req, gate, kms
}

// إنشاء خدمة تحقق بالقيم الافتراضية
func makeVerifyService(kms crypto.KMS, merchantValid bool, txRepo *mockTxRepo) *VerifyService {
	merchant := &mockMerchantVerifier{valid: merchantValid}
	limits := NewLimitsChecker(txRepo, LimitsConfig{
		DailyLimit:   100000,  // 100,000 وحدة صغرى
		MonthlyLimit: 1000000, // 1,000,000 وحدة صغرى
	})
	return NewVerifyService(kms, merchant, limits, 60, 10)
}

// TestVerify_Success — جميع الفحوصات ناجحة
func TestVerify_Success(t *testing.T) {
	req, gate, kms := makeValidRequest()
	txRepo := &mockTxRepo{dailyTotal: 0, monthlyTotal: 0}
	svc := makeVerifyService(kms, true, txRepo)

	result, err := svc.Process(context.Background(), req, gate)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "jawali", result.PayerWalletId)
	assert.Equal(t, "jawali", result.MerchantWalletId)
	assert.Equal(t, int64(2500), result.Amount)
	assert.Equal(t, "YER", result.Currency)
	assert.Equal(t, int64(43), result.NewCounter)
}

// TestVerify_MerchantUnauthorized — رمز وصول التاجر غير صالح
func TestVerify_MerchantUnauthorized(t *testing.T) {
	req, gate, kms := makeValidRequest()
	txRepo := &mockTxRepo{}
	svc := makeVerifyService(kms, false, txRepo) // التاجر غير مُصرَّح

	result, err := svc.Process(context.Background(), req, gate)
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrMerchantUnauthorized, appErr.Code)
	assert.Equal(t, 401, appErr.HTTPStatus)
}

// TestVerify_TimestampExpired — طابع زمني منتهي
func TestVerify_TimestampExpired(t *testing.T) {
	req, gate, kms := makeValidRequest()
	// تعديل الطابع الزمني ليكون قديماً
	req.PaymentToken.Timestamp = time.Now().Unix() - 120 // قبل دقيقتين

	txRepo := &mockTxRepo{}
	svc := makeVerifyService(kms, true, txRepo)

	result, err := svc.Process(context.Background(), req, gate)
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrTimestampExpired, appErr.Code)
	assert.Equal(t, 400, appErr.HTTPStatus)
}

// TestVerify_CounterReplay — عداد إعادة تشغيل (≤ المخزّن)
func TestVerify_CounterReplay(t *testing.T) {
	req, gate, kms := makeValidRequest()
	// العداد أقل من أو يساوي المخزّن
	req.PaymentToken.Counter = gate.PayerCounter // نفس العداد = إعادة تشغيل

	txRepo := &mockTxRepo{}
	svc := makeVerifyService(kms, true, txRepo)

	result, err := svc.Process(context.Background(), req, gate)
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrCounterReplay, appErr.Code)
	assert.Equal(t, 400, appErr.HTTPStatus)
}

// TestVerify_CounterOutOfWindow — عداد خارج نافذة القبول
func TestVerify_CounterOutOfWindow(t *testing.T) {
	req, gate, kms := makeValidRequest()
	// العداد أبعد من النافذة المسموحة (42 + 10 = 52)
	req.PaymentToken.Counter = gate.PayerCounter + 100

	txRepo := &mockTxRepo{}
	svc := makeVerifyService(kms, true, txRepo)

	result, err := svc.Process(context.Background(), req, gate)
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrCounterOutOfWindow, appErr.Code)
	assert.Equal(t, 400, appErr.HTTPStatus)
}

// TestVerify_HMACMismatch — توقيع HMAC غير مطابق
func TestVerify_HMACMismatch(t *testing.T) {
	req, gate, kms := makeValidRequest()
	// تعديل HMAC ليكون خاطئاً
	wrongHMAC := make([]byte, 32)
	rand.Read(wrongHMAC)
	req.PaymentToken.HMAC = base64.StdEncoding.EncodeToString(wrongHMAC)

	txRepo := &mockTxRepo{}
	svc := makeVerifyService(kms, true, txRepo)

	result, err := svc.Process(context.Background(), req, gate)
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrHMACMismatch, appErr.Code)
	assert.Equal(t, 401, appErr.HTTPStatus)
}

// TestVerify_PayerLimitExceeded — المبلغ يتجاوز حد الدافع
func TestVerify_PayerLimitExceeded(t *testing.T) {
	req, gate, kms := makeValidRequest()
	// المبلغ أكبر من حد الدافع
	req.MerchantData.Amount = 10000 // أكبر من PayerLimit = 5000

	txRepo := &mockTxRepo{}
	svc := makeVerifyService(kms, true, txRepo)

	result, err := svc.Process(context.Background(), req, gate)
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrPayerLimitExceeded, appErr.Code)
	assert.Equal(t, 400, appErr.HTTPStatus)
}

// TestVerify_LimitExceeded — تجاوز الحد اليومي/الشهري
func TestVerify_LimitExceeded(t *testing.T) {
	req, gate, kms := makeValidRequest()
	// الإجمالي اليومي قريب من الحد
	txRepo := &mockTxRepo{
		dailyTotal:   99000,  // قريب من الحد اليومي 100,000
		monthlyTotal: 0,
	}
	svc := makeVerifyService(kms, true, txRepo)

	// المبلغ + الإجمالي اليومي > الحد اليومي
	// 99000 + 2500 = 101500 > 100000
	result, err := svc.Process(context.Background(), req, gate)
	require.Error(t, err)
	assert.Nil(t, result)

	var appErr *model.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, model.ErrLimitExceeded, appErr.Code)
	assert.Equal(t, 400, appErr.HTTPStatus)
}
