// عميل HTTP لمحفظة جوالي — إعادة المحاولة + قاطع الدائرة
// يُرجى الرجوع إلى Task 08 — Circuit Breaker
package jawali

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/atheer/switch/internal/adapter"
)

// Client — عميل HTTP لمحفظة جوالي مع إعادة المحاولة وقاطع الدائرة
type Client struct {
	httpClient     *http.Client       // عميل HTTP الأساسي
	baseURL        string             // عنوان API الأساسي
	apiKey         string             // مفتاح API
	secret         string             // السر المشترك
	maxRetries     int                // عدد إعادة المحاولات
	circuitBreaker *adapter.CircuitBreaker // قاطع الدائرة
}

// ClientConfig — إعدادات عميل جوالي
type ClientConfig struct {
	BaseURL   string        // عنوان API الأساسي
	APIKey    string        // مفتاح API
	Secret    string        // السر المشترك
	TimeoutMs int           // مهلة الطلب بالملي ثانية
	MaxRetries int          // عدد إعادة المحاولات
}

// NewClient — ينشئ عميل HTTP جديد لمحفظة جوالي
func NewClient(cfg ClientConfig) *Client {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second // افتراضي
	}
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // افتراضي
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		secret:     cfg.Secret,
		maxRetries: maxRetries,
		circuitBreaker: adapter.NewCircuitBreaker("jawali", 5, 30*time.Second),
	}
}

// CircuitBreaker — يُرجع قاطع الدائرة (للاختبارات)
func (c *Client) CircuitBreaker() *adapter.CircuitBreaker {
	return c.circuitBreaker
}

// post — يرسل طلب POST مع إعادة المحاولة وقاطع الدائرة
// يُرجع جسم الرد أو خطأ
func (c *Client) post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	// 1. التحقق من قاطع الدائرة
	if err := c.circuitBreaker.Allow(); err != nil {
		return nil, err
	}

	// 2. تحويل الجسم إلى JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("جوالي: تحويل JSON: %w", err)
	}

	// 3. إعادة المحاولة
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// انتظار تدريجي قبل إعادة المحاولة
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			slog.Debug("جوالي: إعادة محاولة",
				"path", path,
				"attempt", attempt,
				"backoff_ms", backoff.Milliseconds(),
			)
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("جوالي: السياق أُلغي: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		// إنشاء طلب جديد لكل محاولة (لأن الجسم يُستهلك)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("جوالي: إنشاء طلب: %w", err)
		}

		// تعيين الرؤوس
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", c.apiKey)
		req.Header.Set("X-Secret", c.secret)

		// إرسال الطلب
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("جوالي: فشل الطلب: %w", err)
			slog.Warn("جوالي: فشل الطلب",
				"path", path,
				"attempt", attempt,
				"error", err,
			)
			continue // إعادة المحاولة عند فشل الشبكة
		}

		// قراءة الجسم
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("جوالي: قراءة الرد: %w", err)
			continue
		}

		// التحقق من حالة HTTP
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("جوالي: خطأ خادم %d: %s", resp.StatusCode, string(respBody))
			slog.Warn("جوالي: خطأ خادم",
				"path", path,
				"status", resp.StatusCode,
				"attempt", attempt,
			)
			continue // إعادة المحاولة عند خطأ الخادم
		}

		if resp.StatusCode >= 400 {
			// خطأ العميل — لا إعادة محاولة
			c.circuitBreaker.RecordSuccess() // الخادم استجاب بشكل صحيح
			return respBody, fmt.Errorf("جوالي: خطأ عميل %d: %s", resp.StatusCode, string(respBody))
		}

		// نجاح
		c.circuitBreaker.RecordSuccess()
		return respBody, nil
	}

	// فشلت كل المحاولات
	c.circuitBreaker.RecordFailure()
	return nil, fmt.Errorf("جوالي: فشلت كل المحاولات (%d): %w", c.maxRetries+1, lastErr)
}

// postAndDecode — يرسل طلب POST ويحلّل الرد إلى البنية المحددة
func postAndDecode[Resp any](c *Client, ctx context.Context, path string, body interface{}) (*Resp, error) {
	respBody, err := c.post(ctx, path, body)
	if err != nil {
		return nil, err
	}

	var result Resp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("جوالي: تحليل الرد: %w", err)
	}
	return &result, nil
}
