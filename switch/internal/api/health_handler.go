// معالج فحص الصحة — GET /health
// يُرجع حالة الخادم واتصال قاعدة البيانات وحالة نقطتي الوصول
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler — معالج فحص الصحة
type HealthHandler struct {
	pool          *pgxpool.Pool
	carrierActive bool // هل نقطة وصول الاتصالات مفعّلة
}

// NewHealthHandler — ينشئ معالج فحص الصحة
func NewHealthHandler(pool *pgxpool.Pool, carrierActive bool) *HealthHandler {
	return &HealthHandler{pool: pool, carrierActive: carrierActive}
}

// AccessPointStatus — حالة نقطة الوصول
type AccessPointStatus struct {
	Status  string `json:"status"`  // الحالة: OK أو DISABLED
	Address string `json:"address"` // العنوان مثل :8080
}

// HealthResponse — استجابة فحص الصحة
type HealthResponse struct {
	Status       string            `json:"status"`       // الحالة العامة: OK أو UNHEALTHY
	Timestamp    int64             `json:"timestamp"`    // الطابع الزمني
	Version      string            `json:"version"`      // إصدار الخادم
	DBStatus     string            `json:"dbStatus"`     // حالة قاعدة البيانات
	AccessPoints map[string]AccessPointStatus `json:"accessPoints"` // حالة نقطتي الوصول
}

// Handle — يعالج طلب فحص الصحة
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := HealthResponse{
		Timestamp: time.Now().Unix(),
		Version:   "0.1.0",
		AccessPoints: map[string]AccessPointStatus{
			"internet": {Status: "OK", Address: "public"},
			"carrier":  {Status: "DISABLED", Address: ""},
		},
	}

	if h.carrierActive {
		resp.AccessPoints["carrier"] = AccessPointStatus{Status: "OK", Address: "carrier"}
	}

	// فحص اتصال قاعدة البيانات
	dbStatus := "OK"
	if err := h.pool.Ping(ctx); err != nil {
		dbStatus = "UNHEALTHY"
		resp.Status = "UNHEALTHY"
		slog.Error("فحص الصحة: قاعدة البيانات غير متاحة", "error", err)
	} else {
		resp.Status = "OK"
	}
	resp.DBStatus = dbStatus

	statusCode := http.StatusOK
	if resp.Status == "UNHEALTHY" {
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, resp)
}
