package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/atheer-payment/atheer-platform/pkg/response"
	"github.com/redis/go-redis/v9"
)

// Idempotency is Layer 9 of the transaction pipeline
// Ensures duplicate nonce submissions return the original result (FR-IDEM-001)
// Prevents double-charging on network retries
type Idempotency struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewIdempotency(rdb *redis.Client, ttlSeconds int) *Idempotency {
	return &Idempotency{
		redis: rdb,
		ttl:   time.Duration(ttlSeconds) * time.Second,
	}
}

func (id *Idempotency) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := GetCombinedRequest(r.Context())
		if req == nil {
			response.BadRequest(w, response.ErrInternalError, "Request not parsed")
			return
		}

		nonce := req.SideA.Nonce
		key := fmt.Sprintf("idem:%s", nonce)

		// Check if nonce was already processed
		cached, err := id.redis.Get(r.Context(), key).Bytes()
		if err == nil && len(cached) > 0 {
			// Found cached result — return it
			slog.Info("Idempotency hit — returning cached result",
				"nonce", nonce)

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Idempotent", "true")
			w.WriteHeader(http.StatusOK)
			w.Write(cached)
			return
		}

		// Use a response recorder to capture the downstream response
		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           make([]byte, 0, 1024),
		}

		next.ServeHTTP(rec, r)

		// Cache the result if successful (2xx)
		if rec.statusCode >= 200 && rec.statusCode < 300 {
			cacheData, _ := json.Marshal(map[string]interface{}{
				"cached":  true,
				"nonce":   nonce,
				"result":  json.RawMessage(rec.body),
			})
			if err := id.redis.Set(r.Context(), key, cacheData, id.ttl).Err(); err != nil {
				slog.Warn("Failed to cache idempotency result", "error", err)
			}
		}
	})
}

// CacheResult stores a transaction result for idempotency
func (id *Idempotency) CacheResult(ctx interface{ Deadline() (time.Time, bool) }, nonce string, result interface{}) error {
	key := fmt.Sprintf("idem:%s", nonce)
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return id.redis.Set(ctx.(interface {
		Deadline() (time.Time, bool)
		Done() <-chan struct{}
		Err() error
		Value(interface{}) interface{}
	}), key, data, id.ttl).Err()
}

// responseRecorder captures the response body
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.body = append(rr.body, b...)
	return rr.ResponseWriter.Write(b)
}
