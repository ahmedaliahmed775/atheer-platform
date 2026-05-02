// معالج فحص الصحة — GET /health
// يُرجع حالة الخادم واتصال قاعدة البيانات
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
	pool *pgxpool.Pool
}

// NewHealthHandler — ينشئ معالج فحص الصحة
func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

// HealthResponse — استجابة فحص الصحة
type HealthResponse struct {
	Status    string `json:"status"`    // الحالة: OK أو UNHEALTHY
	Timestamp int64  `json:"timestamp"` // الطابع الزمني
	Version   string `json:"version"`   // إصدار الخادم
	DBStatus  string `json:"dbStatus"`  // حالة قاعدة البيانات
}

// Handle — يعالج طلب فحص الصحة
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := HealthResponse{
		Timestamp: time.Now().Unix(),
		Version:   "0.1.0",
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
