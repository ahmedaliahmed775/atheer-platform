package handler

import (
	"net/http"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/atheer-payment/atheer-platform/pkg/response"
)

var startTime = time.Now()

// HealthHandler handles health check requests
type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

// Check performs health checks on all dependencies
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	health := map[string]interface{}{
		"status":    "healthy",
		"version":   "3.0.0",
		"uptime":    time.Since(startTime).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"go":        runtime.Version(),
	}

	checks := make(map[string]string)

	// Check PostgreSQL
	if err := h.db.Ping(ctx); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		health["status"] = "degraded"
	} else {
		checks["database"] = "healthy"
	}

	// Check Redis
	if err := h.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "unhealthy: " + err.Error()
		health["status"] = "degraded"
	} else {
		checks["redis"] = "healthy"
	}

	health["checks"] = checks

	if health["status"] == "degraded" {
		response.JSON(w, http.StatusServiceUnavailable, health)
		return
	}

	response.OK(w, health)
}
