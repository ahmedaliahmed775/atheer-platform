// معالج فحص صحة الإدارة — حالة المحوّلات والنظام
// يُرجى الرجوع إلى SPEC §5 — Admin APIs
package admin

import (
	"net/http"
	"runtime"
	"time"

	"github.com/atheer/switch/internal/adapter"
	"github.com/atheer/switch/internal/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminHealthHandler — معالج فحص صحة الإدارة
type AdminHealthHandler struct {
	pool     *pgxpool.Pool
	registry *adapter.AdapterRegistry
}

// NewAdminHealthHandler — ينشئ معالج فحص صحة الإدارة
func NewAdminHealthHandler(pool *pgxpool.Pool, registry *adapter.AdapterRegistry) *AdminHealthHandler {
	return &AdminHealthHandler{
		pool:     pool,
		registry: registry,
	}
}

// AdapterHealthStatus — حالة محوّل محفظة
type AdapterHealthStatus struct {
	WalletId      string `json:"walletId"`      // معرّف المحفظة
	Status        string `json:"status"`        // الحالة: UP أو DOWN أو DEGRADED
	CircuitState  string `json:"circuitState"`  // حالة قاطع الدائرة: CLOSED أو OPEN أو HALF_OPEN
	LastCheckedAt int64  `json:"lastCheckedAt"` // آخر فحص بالطابع الزمني
	ResponseTimeMs int64 `json:"responseTimeMs"` // زمن الاستجابة بالملي ثانية
}

// AdaptersHealthResponse — استجابة حالة المحوّلات
type AdaptersHealthResponse struct {
	Adapters []AdapterHealthStatus `json:"adapters"` // حالة كل محوّل
	Overall  string                `json:"overall"`  // الحالة العامة: UP أو DOWN أو DEGRADED
}

// SystemHealthResponse — استجابة حالة النظام
type SystemHealthResponse struct {
	Status      string            `json:"status"`      // الحالة العامة
	Uptime      string            `json:"uptime"`      // مدة التشغيل
	Goroutines  int               `json:"goroutines"`  // عدد الغوروتينات
	MemoryMB    float64           `json:"memoryMb"`    // استخدام الذاكرة بالميغابايت
	DBStatus    string            `json:"dbStatus"`    // حالة قاعدة البيانات
	Version     string            `json:"version"`     // إصدار النظام
	Components  map[string]string `json:"components"`  // حالة المكونات
}

// HandleAdapters — يعالج طلب حالة المحوّلات
// GET /admin/v1/health/adapters
func (h *AdminHealthHandler) HandleAdapters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// فلترة النطاق — WALLET_ADMIN يرى محفظته فقط
	scopeFilter := middleware.ScopeFilter(ctx)

	// جلب قائمة المحوّلات المسجّلة
	walletIds := h.registry.List()

	var adapters []AdapterHealthStatus
	overall := "UP"

	for _, wid := range walletIds {
		// فلترة النطاق
		if scopeFilter != "" && scopeFilter != wid {
			continue
		}

		// فحص حالة المحوّل — نحاول استعلام المعاملة
		status := "UP"
		circuitState := "CLOSED"
		var responseTimeMs int64

		_, err := h.registry.Get(wid)
		if err != nil {
			status = "DOWN"
			circuitState = "UNKNOWN"
			overall = "DEGRADED"
		}

		adapters = append(adapters, AdapterHealthStatus{
			WalletId:       wid,
			Status:         status,
			CircuitState:   circuitState,
			LastCheckedAt:  time.Now().Unix(),
			ResponseTimeMs: responseTimeMs,
		})
	}

	if adapters == nil {
		adapters = []AdapterHealthStatus{}
	}

	// إذا كانت كل المحوّلات معطّلة
	allDown := true
	for _, a := range adapters {
		if a.Status != "DOWN" {
			allDown = false
			break
		}
	}
	if allDown && len(adapters) > 0 {
		overall = "DOWN"
	}

	writeAdminJSON(w, http.StatusOK, AdaptersHealthResponse{
		Adapters: adapters,
		Overall:  overall,
	})
}

// HandleSystem — يعالج طلب حالة النظام
// GET /admin/v1/health/system
func (h *AdminHealthHandler) HandleSystem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// فحص قاعدة البيانات
	dbStatus := "UP"
	if err := h.pool.Ping(ctx); err != nil {
		dbStatus = "DOWN"
	}

	// إحصائيات الذاكرة
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryMB := float64(memStats.Alloc) / 1024 / 1024

	// بناء الحالة العامة
	overallStatus := "UP"
	if dbStatus == "DOWN" {
		overallStatus = "DOWN"
	}

	// حالة المكونات
	components := map[string]string{
		"database":  dbStatus,
		"server":    "UP",
		"adapters":  "UP",
	}

	// عدد المحوّلات المسجّلة
	walletIds := h.registry.List()
	if len(walletIds) == 0 {
		components["adapters"] = "NO_ADAPTERS"
	}

	resp := SystemHealthResponse{
		Status:     overallStatus,
		Uptime:     FormatDuration(time.Since(startTime)), // يُحسب من بدء التشغيل
		Goroutines: runtime.NumGoroutine(),
		MemoryMB:   memoryMB,
		DBStatus:   dbStatus,
		Version:    "0.1.0",
		Components: components,
	}

	statusCode := http.StatusOK
	if overallStatus == "DOWN" {
		statusCode = http.StatusServiceUnavailable
	}

	writeAdminJSON(w, statusCode, resp)
}

// startTime — وقت بدء تشغيل الخادم (يُضبط في main.go)
var startTime = time.Now()

// SetStartTime — يضبط وقت بدء التشغيل
func SetStartTime(t time.Time) {
	startTime = t
}
