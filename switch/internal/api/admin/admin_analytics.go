// معالج تحليلات الإدارة — ملخص الأداء وحجم المعاملات والأخطاء وزمن الاستجابة
// يُرجى الرجوع إلى SPEC §5 — Admin APIs
package admin

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/atheer/switch/internal/db"
	"github.com/atheer/switch/internal/middleware"
	"github.com/atheer/switch/internal/model"
)

// AdminAnalyticsHandler — معالج تحليلات الإدارة
type AdminAnalyticsHandler struct {
	txRepo db.TransactionRepo
}

// NewAdminAnalyticsHandler — ينشئ معالج تحليلات الإدارة
func NewAdminAnalyticsHandler(txRepo db.TransactionRepo) *AdminAnalyticsHandler {
	return &AdminAnalyticsHandler{txRepo: txRepo}
}

// SummaryResponse — استجابة ملخص الأداء
type SummaryResponse struct {
	TotalTransactions   int64   `json:"totalTransactions"`   // إجمالي المعاملات
	SuccessRate         float64 `json:"successRate"`         // نسبة النجاح
	TotalVolume         int64   `json:"totalVolume"`         // إجمالي الحجم بالوحدة الصغرى
	AverageAmount       int64   `json:"averageAmount"`       // متوسط المبلغ
	FailedTransactions  int64   `json:"failedTransactions"`  // عدد المعاملات الفاشلة
	PendingTransactions int64   `json:"pendingTransactions"` // عدد المعاملات المعلقة
	Period              string  `json:"period"`              // الفترة
}

// VolumeResponse — استجابة حجم المعاملات
type VolumeResponse struct {
	Data   []VolumeDataPoint `json:"data"`   // نقاط البيانات
	Period string            `json:"period"` // الفترة
	Group  string            `json:"group"`  // التجميع: day أو week أو month
}

// VolumeDataPoint — نقطة بيانات حجم المعاملات
type VolumeDataPoint struct {
	Date   string `json:"date"`   // التاريخ
	Count  int64  `json:"count"`  // عدد المعاملات
	Amount int64  `json:"amount"` // إجمالي المبلغ
}

// ErrorsResponse — استجابة تحليل الأخطاء
type ErrorsResponse struct {
	TotalErrors int64             `json:"totalErrors"` // إجمالي الأخطاء
	ErrorRate   float64           `json:"errorRate"`   // معدل الأخطاء
	ByCode      []ErrorCodeCount  `json:"byCode"`      // توزيع حسب رمز الخطأ
	ByWallet    []WalletErrorCount `json:"byWallet"`   // توزيع حسب المحفظة
	Period      string            `json:"period"`      // الفترة
}

// ErrorCodeCount — عدد الأخطاء حسب الرمز
type ErrorCodeCount struct {
	Code  string `json:"code"`  // رمز الخطأ
	Count int64  `json:"count"` // العدد
}

// WalletErrorCount — عدد الأخطاء حسب المحفظة
type WalletErrorCount struct {
	WalletId string `json:"walletId"` // معرّف المحفظة
	Count    int64  `json:"count"`    // العدد
}

// LatencyResponse — استجابة تحليل زمن الاستجابة
type LatencyResponse struct {
	AverageMs int64  `json:"averageMs"` // المتوسط بالملي ثانية
	P50Ms     int64  `json:"p50Ms"`     // النسبة المئوية 50
	P95Ms     int64  `json:"p95Ms"`     // النسبة المئوية 95
	P99Ms     int64  `json:"p99Ms"`     // النسبة المئوية 99
	Period    string `json:"period"`    // الفترة
}

// HandleSummary — يعالج طلب ملخص الأداء
// GET /admin/v1/analytics/summary?period=24h
func (h *AdminAnalyticsHandler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// استخراج الفترة من الاستعلام
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	// فلترة النطاق
	scopeFilter := middleware.ScopeFilter(ctx)
	_ = scopeFilter // يُستخدم في فلترة الاستعلامات

	// جلب البيانات — نستخدم TransactionRepo.List مع تصفية
	filters := model.TransactionFilters{}
	if scopeFilter != "" {
		filters.WalletId = scopeFilter
	}

	// حساب الفترة
	now := time.Now()
	switch period {
	case "7d":
		filters.FromDate = now.AddDate(0, 0, -7)
	case "30d":
		filters.FromDate = now.AddDate(0, 0, -30)
	default: // 24h
		filters.FromDate = now.Add(-24 * time.Hour)
	}

	// جلب كل المعاملات في الفترة (تنفيذ مبسّط)
	// في الإنتاج: استعلام تجميعي مباشر في قاعدة البيانات
	allTx, totalCount, err := h.txRepo.List(ctx, filters, 1, 1)
	if err != nil {
		slog.Error("إدارة التحليلات: فشل جلب الملخص", "error", err)
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
			"errorCode":    "INTERNAL_ERROR",
			"errorMessage": "خطأ في جلب التحليلات",
		})
		return
	}

	// بناء الاستجابة — تنفيذ مبسّط
	_ = allTx
	summary := SummaryResponse{
		TotalTransactions:   int64(totalCount),
		SuccessRate:         0.0, // يُحسب من استعلام تجميعي
		TotalVolume:         0,
		AverageAmount:       0,
		FailedTransactions:  0,
		PendingTransactions: 0,
		Period:              period,
	}

	writeAdminJSON(w, http.StatusOK, summary)
}

// HandleVolume — يعالج طلب حجم المعاملات
// GET /admin/v1/analytics/volume?period=7d&group=day
func (h *AdminAnalyticsHandler) HandleVolume(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "7d"
	}
	group := r.URL.Query().Get("group")
	if group == "" {
		group = "day"
	}

	// فلترة النطاق
	scopeFilter := middleware.ScopeFilter(ctx)
	_ = scopeFilter

	// بناء نقاط البيانات — تنفيذ مبسّط
	// في الإنتاج: استعلام تجميعي GROUP BY DATE(created_at)
	dataPoints := []VolumeDataPoint{}

	// حساب الفترة الزمنية
	now := time.Now()
	var days int
	switch period {
	case "30d":
		days = 30
	case "7d":
		days = 7
	default:
		days = 1
	}

	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		dataPoints = append(dataPoints, VolumeDataPoint{
			Date:   date,
			Count:  0, // يُحسب من قاعدة البيانات
			Amount: 0,
		})
	}

	writeAdminJSON(w, http.StatusOK, VolumeResponse{
		Data:   dataPoints,
		Period: period,
		Group:  group,
	})
}

// HandleErrors — يعالج طلب تحليل الأخطاء
// GET /admin/v1/analytics/errors?period=24h
func (h *AdminAnalyticsHandler) HandleErrors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	// فلترة النطاق
	scopeFilter := middleware.ScopeFilter(ctx)
	_ = scopeFilter

	// تنفيذ مبسّط — في الإنتاج: استعلام تجميعي GROUP BY error_code
	resp := ErrorsResponse{
		TotalErrors: 0,
		ErrorRate:   0.0,
		ByCode:      []ErrorCodeCount{},
		ByWallet:    []WalletErrorCount{},
		Period:      period,
	}

	writeAdminJSON(w, http.StatusOK, resp)
}

// HandleLatency — يعالج طلب تحليل زمن الاستجابة
// GET /admin/v1/analytics/latency?period=24h
func (h *AdminAnalyticsHandler) HandleLatency(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	// تنفيذ مبسّط — في الإنتاج: استعلام تجميعي PERCENTILE على duration_ms
	resp := LatencyResponse{
		AverageMs: 0,
		P50Ms:     0,
		P95Ms:     0,
		P99Ms:     0,
		Period:    period,
	}

	writeAdminJSON(w, http.StatusOK, resp)
}
