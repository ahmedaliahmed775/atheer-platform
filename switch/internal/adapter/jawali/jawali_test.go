// اختبارات محوّل محفظة جوالي — تستخدم httptest.NewServer
// يُرجى الرجوع إلى Task 08 — Tests
package jawali

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	adapterpkg "github.com/atheer/switch/internal/adapter"
	"github.com/atheer/switch/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- اختبار الخصم ---

// TestJawali_Debit_Success — اختبار خصم ناجح
func TestJawali_Debit_Success(t *testing.T) {
	// إعداد خادم اختبار
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// التحقق من المسار والطريقة
		assert.Equal(t, "/cashout", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// تحليل الطلب
		var req JawaliCashoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("فشل تحليل الطلب: %v", err)
		}
		assert.Equal(t, "777123456", req.PayerPhone)
		assert.Equal(t, int64(50000), req.Amount)
		assert.Equal(t, "YER", req.Currency)

		// إرجاع رد ناجح
		resp := JawaliCashoutResponse{
			JawaliResponse: JawaliResponse{
				ResponseCode:    ResponseCodeSuccess,
				ResponseMessage: "Success",
				Reference:       req.Reference,
				Status:          "SUCCESS",
			},
			TransactionRef: "JWL-TXN-001",
			Balance:        100000,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// إنشاء المحوّل
	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  5000,
		MaxRetries: 1,
	})

	// تنفيذ الخصم
	result, err := adapter.Debit(context.Background(), model.DebitParams{
		WalletId:      "777123456",
		AccountRef:    "merchant-001",
		Amount:        50000,
		Currency:      "YER",
		IdempotencyKey: "idem-001",
	})

	// التحقق من النتيجة
	require.NoError(t, err)
	assert.Equal(t, "JWL-TXN-001", result.DebitRef)
	assert.Equal(t, "SUCCESS", result.Status)
}

// TestJawali_Debit_InsufficientFunds — اختبار خصم برصيد غير كافٍ
func TestJawali_Debit_InsufficientFunds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := JawaliCashoutResponse{
			JawaliResponse: JawaliResponse{
				ResponseCode:    ResponseCodeInsufficientFunds,
				ResponseMessage: "Insufficient funds",
				Status:          "FAILED",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  5000,
		MaxRetries: 1,
	})

	result, err := adapter.Debit(context.Background(), model.DebitParams{
		WalletId:      "777123456",
		AccountRef:    "merchant-001",
		Amount:        999999,
		Currency:      "YER",
		IdempotencyKey: "idem-002",
	})

	// الخصم لا يُرجع خطأ — يُرجع نتيجة بحالة FAILED
	require.NoError(t, err)
	assert.Equal(t, "FAILED", result.Status)
}

// --- اختبار الإيداع ---

// TestJawali_Credit_Success — اختبار إيداع ناجح
func TestJawali_Credit_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/cashin", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var req JawaliCashinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("فشل تحليل الطلب: %v", err)
		}
		assert.Equal(t, "merchant-001", req.MerchantId)
		assert.Equal(t, int64(50000), req.Amount)

		resp := JawaliCashinResponse{
			JawaliResponse: JawaliResponse{
				ResponseCode:    ResponseCodeSuccess,
				ResponseMessage: "Success",
				Reference:       req.Reference,
				Status:          "SUCCESS",
			},
			TransactionRef: "JWL-CIN-001",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  5000,
		MaxRetries: 1,
	})

	result, err := adapter.Credit(context.Background(), model.CreditParams{
		WalletId:      "jawali",
		AccountRef:    "merchant-001",
		Amount:        50000,
		Currency:      "YER",
		IdempotencyKey: "idem-003",
	})

	require.NoError(t, err)
	assert.Equal(t, "JWL-CIN-001", result.CreditRef)
	assert.Equal(t, "SUCCESS", result.Status)
}

// --- اختبار التحقق من الرمز ---

// TestJawali_VerifyAccessToken_Success — اختبار تحقق ناجح من رمز الوصول
func TestJawali_VerifyAccessToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/verify", r.URL.Path)

		resp := JawaliAuthVerifyResponse{
			JawaliResponse: JawaliResponse{
				ResponseCode:    ResponseCodeSuccess,
				ResponseMessage: "Valid token",
				Status:          "SUCCESS",
			},
			Valid: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  5000,
		MaxRetries: 1,
	})

	valid, err := adapter.VerifyAccessToken(context.Background(), "jawali", "valid-token")
	require.NoError(t, err)
	assert.True(t, valid)
}

// TestJawali_VerifyAccessToken_Invalid — اختبار رمز وصول غير صالح
func TestJawali_VerifyAccessToken_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := JawaliAuthVerifyResponse{
			JawaliResponse: JawaliResponse{
				ResponseCode:    ResponseCodeSuccess,
				ResponseMessage: "Invalid token",
				Status:          "FAILED",
			},
			Valid: false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  5000,
		MaxRetries: 1,
	})

	valid, err := adapter.VerifyAccessToken(context.Background(), "jawali", "bad-token")
	require.NoError(t, err)
	assert.False(t, valid)
}

// --- اختبار عكس الخصم ---

// TestJawali_ReverseDebit_Success — اختبار عكس خصم ناجح
func TestJawali_ReverseDebit_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/cashout/reverse", r.URL.Path)

		resp := JawaliReverseResponse{
			JawaliResponse: JawaliResponse{
				ResponseCode:    ResponseCodeSuccess,
				ResponseMessage: "Reversed",
				Status:          "SUCCESS",
			},
			ReverseRef: "JWL-REV-001",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  5000,
		MaxRetries: 1,
	})

	result, err := adapter.ReverseDebit(context.Background(), "JWL-TXN-001")
	require.NoError(t, err)
	assert.Equal(t, "JWL-REV-001", result.ReverseRef)
	assert.Equal(t, "SUCCESS", result.Status)
}

// --- اختبار الاستعلام ---

// TestJawali_QueryTransaction_Success — اختبار استعلام ناجح
func TestJawali_QueryTransaction_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/inquiry", r.URL.Path)

		resp := JawaliInquiryResponse{
			JawaliResponse: JawaliResponse{
				ResponseCode:    ResponseCodeSuccess,
				ResponseMessage: "Found",
				Status:          "SUCCESS",
			},
			TransactionRef: "JWL-TXN-001",
			Amount:         50000,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  5000,
		MaxRetries: 1,
	})

	result, err := adapter.QueryTransaction(context.Background(), "JWL-TXN-001")
	require.NoError(t, err)
	assert.Equal(t, "JWL-TXN-001", result.Ref)
	assert.Equal(t, "SUCCESS", result.Status)
}

// --- اختبار قاطع الدائرة ---

// TestJawali_CircuitBreaker_Opens — اختبار فتح قاطع الدائرة بعد 5 فشل متتالي
func TestJawali_CircuitBreaker_Opens(t *testing.T) {
	var requestCount int32

	// خادم يُرجع خطأ 500 دائماً
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"responseCode":"999","responseMessage":"Internal error"}`))
	}))
	defer server.Close()

	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  2000,
		MaxRetries: 0, // بدون إعادة محاولة لتسريع الاختبار
	})

	cb := adapter.client.CircuitBreaker()

	// تنفيذ 5 طلبات فاشلة لفتح قاطع الدائرة
	for i := 0; i < 5; i++ {
		_, err := adapter.Debit(context.Background(), model.DebitParams{
			WalletId:      "777123456",
			AccountRef:    "merchant-001",
			Amount:        1000,
			Currency:      "YER",
			IdempotencyKey: "idem-cb",
		})
		require.Error(t, err, "الطلب %d يجب أن يفشل", i+1)
	}

	// قاطع الدائرة يجب أن يكون في حالة OPEN
	assert.Equal(t, adapterpkg.Open, cb.State(), "قاطع الدائرة يجب أن يكون OPEN بعد 5 فشل")

	// الطلب التالي يجب أن يُرفض فوراً بدون الاتصال بالخادم
	prevCount := atomic.LoadInt32(&requestCount)
	_, err := adapter.Debit(context.Background(), model.DebitParams{
		WalletId:      "777123456",
		AccountRef:    "merchant-001",
		Amount:        1000,
		Currency:      "YER",
		IdempotencyKey: "idem-cb-blocked",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OPEN", "الخطأ يجب أن يُشير إلى حالة OPEN")

	// عدد الطلبات للخادم لم يتغير (الطلب رُفض محلياً)
	assert.Equal(t, prevCount, atomic.LoadInt32(&requestCount),
		"لا يجب إرسال طلب جديد عندما قاطع الدائرة OPEN")
}

// --- اختبار إعادة المحاولة ---

// TestJawali_Retry_OnServerError — اختبار إعادة المحاولة عند خطأ الخادم
func TestJawali_Retry_OnServerError(t *testing.T) {
	var attemptCount int32

	// خادم يفشل مرتين ثم ينجح
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"responseCode":"999","responseMessage":"Temporary error"}`))
			return
		}

		resp := JawaliCashoutResponse{
			JawaliResponse: JawaliResponse{
				ResponseCode:    ResponseCodeSuccess,
				ResponseMessage: "Success",
				Status:          "SUCCESS",
			},
			TransactionRef: "JWL-TXN-RETRY",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  5000,
		MaxRetries: 3,
	})

	result, err := adapter.Debit(context.Background(), model.DebitParams{
		WalletId:      "777123456",
		AccountRef:    "merchant-001",
		Amount:        1000,
		Currency:      "YER",
		IdempotencyKey: "idem-retry",
	})

	require.NoError(t, err)
	assert.Equal(t, "JWL-TXN-RETRY", result.DebitRef)
	assert.Equal(t, "SUCCESS", result.Status)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount), "يجب المحاولة 3 مرات")
}

// --- اختبار إلغاء السياق ---

// TestJawali_ContextCancellation — اختبار إلغاء السياق
func TestJawali_ContextCancellation(t *testing.T) {
	// خادم بطيء
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewJawaliAdapter(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		Secret:     "test-secret",
		TimeoutMs:  10000,
		MaxRetries: 0,
	})

	// سياق مع مهلة قصيرة
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := adapter.Debit(ctx, model.DebitParams{
		WalletId:      "777123456",
		AccountRef:    "merchant-001",
		Amount:        1000,
		Currency:      "YER",
		IdempotencyKey: "idem-cancel",
	})

	require.Error(t, err)
	assert.ErrorIs(t, context.DeadlineExceeded, ctx.Err())
}
