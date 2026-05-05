// اختبارات تكاملية لمعالجات HTTP — تسجيل مستخدم + معاملة + أخطاء
// يستخدم mock repos بدون قاعدة بيانات حقيقية
package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atheer/switch/internal/api"
	"github.com/atheer/switch/internal/crypto"
	"github.com/atheer/switch/internal/model"
)

// ════════════════════════════════════════
// Mock Repos
// ════════════════════════════════════════

// mockPayerRepo — تنفيذ وهمي لمستودع الدافعين
type mockPayerRepo struct {
	records map[string]*model.SwitchRecord
}

func newMockPayerRepo() *mockPayerRepo {
	return &mockPayerRepo{records: make(map[string]*model.SwitchRecord)}
}

func (m *mockPayerRepo) FindByPublicId(_ context.Context, publicId string) (*model.SwitchRecord, error) {
	r, ok := m.records[publicId]
	if !ok {
		return nil, nil
	}
	return r, nil
}

func (m *mockPayerRepo) Create(_ context.Context, record *model.SwitchRecord) error {
	// التحقق من تكرار الجهاز
	for _, r := range m.records {
		if r.DeviceId == record.DeviceId {
			return &duplicateError{}
		}
	}
	m.records[record.PublicId] = record
	return nil
}

func (m *mockPayerRepo) UpdateCounter(_ context.Context, publicId string, newCounter int64) error {
	if r, ok := m.records[publicId]; ok {
		r.Counter = newCounter
	}
	return nil
}

func (m *mockPayerRepo) UpdateStatus(_ context.Context, publicId string, status string) error {
	if r, ok := m.records[publicId]; ok {
		r.Status = status
	}
	return nil
}

func (m *mockPayerRepo) UpdatePayerLimit(_ context.Context, publicId string, newLimit int64) error {
	if r, ok := m.records[publicId]; ok {
		r.PayerLimit = newLimit
	}
	return nil
}

func (m *mockPayerRepo) Delete(_ context.Context, publicId string) error {
	delete(m.records, publicId)
	return nil
}

type duplicateError struct{}

func (e *duplicateError) Error() string { return "uq_device_id" }

// mockWalletRepo — تنفيذ وهمي لمستودع المحافظ
type mockWalletRepo struct {
	wallets map[string]*model.WalletConfig
}

func newMockWalletRepo() *mockWalletRepo {
	return &mockWalletRepo{wallets: map[string]*model.WalletConfig{
		"jawali": {
			ID:            1,
			WalletId:      "jawali",
			BaseURL:       "https://api.jawali.ye",
			APIKey:        "test-key",
			Secret:        "test-secret",
			MaxPayerLimit: 5000000, // 50,000 YER
			TimeoutMs:     10000,
			MaxRetries:    2,
			IsActive:      true,
		},
		"floosak": {
			ID:            2,
			WalletId:      "floosak",
			BaseURL:       "https://api.floosak.ye",
			MaxPayerLimit: 3000000,
			TimeoutMs:     10000,
			MaxRetries:    2,
			IsActive:      false, // معطّلة
		},
	}}
}

func (m *mockWalletRepo) FindByWalletId(_ context.Context, walletId string) (*model.WalletConfig, error) {
	w, ok := m.wallets[walletId]
	if !ok {
		return nil, nil
	}
	return w, nil
}

func (m *mockWalletRepo) List(_ context.Context) ([]model.WalletConfig, error) {
	var result []model.WalletConfig
	for _, w := range m.wallets {
		result = append(result, *w)
	}
	return result, nil
}

func (m *mockWalletRepo) Create(_ context.Context, config *model.WalletConfig) error {
	m.wallets[config.WalletId] = config
	return nil
}

func (m *mockWalletRepo) Update(_ context.Context, config *model.WalletConfig) error {
	m.wallets[config.WalletId] = config
	return nil
}

// mockKMS — تنفيذ وهمي لنظام إدارة المفاتيح
type mockKMS struct{}

func (m *mockKMS) Encrypt(_ context.Context, plaintext []byte) ([]byte, string, error) {
	// تشفير بسيط: قلب البايتات
	enc := make([]byte, len(plaintext))
	for i, b := range plaintext {
		enc[len(plaintext)-1-i] = b
	}
	return enc, "test-key-id", nil
}

func (m *mockKMS) Decrypt(_ context.Context, _ string, ciphertext []byte) ([]byte, error) {
	dec := make([]byte, len(ciphertext))
	for i, b := range ciphertext {
		dec[len(ciphertext)-1-i] = b
	}
	return dec, nil
}

// تأكد أن mockKMS يُلبّي واجهة KMS
var _ crypto.KMS = (*mockKMS)(nil)

// ════════════════════════════════════════
// دوال مساعدة
// ════════════════════════════════════════

func makeJSON(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		t.Fatalf("فشل ترميز JSON: %v", err)
	}
	return buf
}

type errorResponse struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// ════════════════════════════════════════
// اختبارات تسجيل المستخدم (Enroll)
// ════════════════════════════════════════

func TestEnroll_Success(t *testing.T) {
	payerRepo := newMockPayerRepo()
	walletRepo := newMockWalletRepo()
	kms := &mockKMS{}
	handler := api.NewEnrollHandler(payerRepo, walletRepo, kms)

	body := makeJSON(t, map[string]string{
		"walletId":    "jawali",
		"walletToken": "valid-token",
		"deviceId":    "device-001",
		"userType":    "P",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("التسجيل: توقّعنا 201، حصلنا على %d: %s", w.Code, w.Body.String())
	}

	var resp model.EnrollResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("فشل تحليل الاستجابة: %v", err)
	}
	if resp.PublicId == "" {
		t.Error("التسجيل: المعرّف العام فارغ")
	}
	if resp.Status != "ACTIVE" {
		t.Errorf("التسجيل: توقّعنا ACTIVE، حصلنا على %s", resp.Status)
	}
	if resp.EncryptedSeed == "" {
		t.Error("التسجيل: البذرة المشفّرة فارغة")
	}
	if resp.PayerLimit != 5000000 {
		t.Errorf("التسجيل: توقّعنا حد 5000000، حصلنا على %d", resp.PayerLimit)
	}
}

func TestEnroll_MissingFields(t *testing.T) {
	handler := api.NewEnrollHandler(newMockPayerRepo(), newMockWalletRepo(), &mockKMS{})

	cases := []struct {
		name string
		body map[string]string
	}{
		{"بدون walletId", map[string]string{"walletToken": "t", "deviceId": "d", "userType": "P"}},
		{"بدون walletToken", map[string]string{"walletId": "jawali", "deviceId": "d", "userType": "P"}},
		{"بدون deviceId", map[string]string{"walletId": "jawali", "walletToken": "t", "userType": "P"}},
		{"userType غير صالح", map[string]string{"walletId": "jawali", "walletToken": "t", "deviceId": "d", "userType": "X"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", makeJSON(t, tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.Handle(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: توقّعنا 400، حصلنا على %d", tc.name, w.Code)
			}
		})
	}
}

func TestEnroll_WalletNotFound(t *testing.T) {
	handler := api.NewEnrollHandler(newMockPayerRepo(), newMockWalletRepo(), &mockKMS{})

	body := makeJSON(t, map[string]string{
		"walletId": "unknown_wallet", "walletToken": "t",
		"deviceId": "d", "userType": "P",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("محفظة غير موجودة: توقّعنا 404، حصلنا على %d", w.Code)
	}
	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ErrorCode != model.ErrWalletNotFound {
		t.Errorf("رمز خطأ: توقّعنا %s، حصلنا على %s", model.ErrWalletNotFound, resp.ErrorCode)
	}
}

func TestEnroll_WalletInactive(t *testing.T) {
	handler := api.NewEnrollHandler(newMockPayerRepo(), newMockWalletRepo(), &mockKMS{})

	body := makeJSON(t, map[string]string{
		"walletId": "floosak", "walletToken": "t",
		"deviceId": "d", "userType": "P",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("محفظة معطّلة: توقّعنا 403، حصلنا على %d", w.Code)
	}
	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ErrorCode != model.ErrWalletInactive {
		t.Errorf("رمز خطأ: توقّعنا %s، حصلنا على %s", model.ErrWalletInactive, resp.ErrorCode)
	}
}

func TestEnroll_DuplicateDevice(t *testing.T) {
	payerRepo := newMockPayerRepo()
	handler := api.NewEnrollHandler(payerRepo, newMockWalletRepo(), &mockKMS{})

	body := map[string]string{
		"walletId": "jawali", "walletToken": "t",
		"deviceId": "same-device", "userType": "P",
	}

	// التسجيل الأول — يجب أن ينجح
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", makeJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Handle(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("التسجيل الأول: توقّعنا 201، حصلنا على %d", w.Code)
	}

	// التسجيل الثاني بنفس الجهاز — يجب أن يفشل
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", makeJSON(t, body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.Handle(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Errorf("تكرار الجهاز: توقّعنا 409، حصلنا على %d", w2.Code)
	}
}

func TestEnroll_InvalidJSON(t *testing.T) {
	handler := api.NewEnrollHandler(newMockPayerRepo(), newMockWalletRepo(), &mockKMS{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/enroll", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("JSON غير صالح: توقّعنا 400، حصلنا على %d", w.Code)
	}
}

// ════════════════════════════════════════
// اختبارات المعاملة (Transaction)
// ════════════════════════════════════════

func TestTransaction_MissingPaymentToken(t *testing.T) {
	handler := api.NewTransactionHandler(nil, nil, nil)

	body := makeJSON(t, map[string]interface{}{
		"paymentToken": map[string]interface{}{
			"publicId": "",
			"deviceId": "",
		},
		"merchantData": map[string]interface{}{
			"merchantId":       "m1",
			"merchantWalletId": "jawali",
			"amount":           1000,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("حقول ناقصة: توقّعنا 400، حصلنا على %d: %s", w.Code, w.Body.String())
	}
}

func TestTransaction_MissingMerchantData(t *testing.T) {
	handler := api.NewTransactionHandler(nil, nil, nil)

	body := makeJSON(t, map[string]interface{}{
		"paymentToken": map[string]interface{}{
			"publicId": "usr_abc123",
			"deviceId": "dev_456",
		},
		"merchantData": map[string]interface{}{
			"merchantId":       "",
			"merchantWalletId": "",
			"amount":           0,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("بيانات التاجر ناقصة: توقّعنا 400، حصلنا على %d: %s", w.Code, w.Body.String())
	}
}

func TestTransaction_InvalidJSON(t *testing.T) {
	handler := api.NewTransactionHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transaction", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("JSON غير صالح: توقّعنا 400، حصلنا على %d", w.Code)
	}
}
