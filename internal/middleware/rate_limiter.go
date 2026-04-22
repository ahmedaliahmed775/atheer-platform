package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/atheer-payment/atheer-platform/pkg/response"
	"github.com/redis/go-redis/v9"
)

// RateLimiter implements tiered rate limiting per device, wallet, and IP
// Layer 1 of the transaction pipeline (FR-SEC-003)
type RateLimiter struct {
	redis          *redis.Client
	perDeviceLimit int
	perWalletLimit int
	perIPLimit     int
	window         time.Duration
	// In-memory fallback if Redis is unavailable
	mu       sync.Mutex
	counters map[string]*rateBucket
}

type rateBucket struct {
	count    int
	windowAt time.Time
}

// NewRateLimiter creates a new tiered rate limiter
func NewRateLimiter(rdb *redis.Client, perDevice, perWallet, perIP int) *RateLimiter {
	return &RateLimiter{
		redis:          rdb,
		perDeviceLimit: perDevice,
		perWalletLimit: perWallet,
		perIPLimit:     perIP,
		window:         time.Minute,
		counters:       make(map[string]*rateBucket),
	}
}

// Middleware returns the Chi middleware handler
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract identifiers from request
		deviceID := r.Header.Get("X-Device-ID")
		walletID := r.Header.Get("X-Wallet-ID")
		ip := r.RemoteAddr

		// Check all three tiers
		checks := []struct {
			key   string
			limit int
			label string
		}{
			{fmt.Sprintf("rl:dev:%s", deviceID), rl.perDeviceLimit, "device"},
			{fmt.Sprintf("rl:wal:%s", walletID), rl.perWalletLimit, "wallet"},
			{fmt.Sprintf("rl:ip:%s", ip), rl.perIPLimit, "ip"},
		}

		for _, check := range checks {
			if check.key == "rl:dev:" || check.key == "rl:wal:" {
				continue // Skip if no identifier
			}

			count, err := rl.redis.Incr(ctx, check.key).Result()
			if err != nil {
				// Fallback to in-memory
				if rl.checkInMemory(check.key, check.limit) {
					slog.Warn("Rate limit exceeded (in-memory fallback)",
						"tier", check.label, "key", check.key)
					response.TooManyRequests(w)
					return
				}
				continue
			}

			if count == 1 {
				rl.redis.Expire(ctx, check.key, rl.window)
			}

			if int(count) > check.limit {
				slog.Warn("Rate limit exceeded",
					"tier", check.label,
					"count", count,
					"limit", check.limit,
				)
				response.TooManyRequests(w)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) checkInMemory(key string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.counters[key]
	if !exists || now.Sub(bucket.windowAt) > rl.window {
		rl.counters[key] = &rateBucket{count: 1, windowAt: now}
		return false
	}
	bucket.count++
	return bucket.count > limit
}
