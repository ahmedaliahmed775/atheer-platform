package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/atheer-payment/atheer-platform/internal/model"
	"github.com/atheer-payment/atheer-platform/pkg/response"
	"github.com/redis/go-redis/v9"
)

// AntiReplay is Layer 3 of the transaction pipeline
// Uses Redis Lua script for atomic counter check (FR-SEC-002)
// Ensures ctr(new) > ctr(stored) — strictly increasing
type AntiReplay struct {
	redis *redis.Client
	// Lua script loaded once
	script *redis.Script
}

const antiReplayLua = `
local key = KEYS[1]
local newCtr = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2]) or 86400

local storedCtr = tonumber(redis.call('GET', key))

if storedCtr == nil then
    redis.call('SETEX', key, ttl, newCtr)
    return 1
end

if newCtr > storedCtr then
    redis.call('SETEX', key, ttl, newCtr)
    return 1
else
    return 0
end
`

func NewAntiReplay(rdb *redis.Client) *AntiReplay {
	return &AntiReplay{
		redis:  rdb,
		script: redis.NewScript(antiReplayLua),
	}
}

func (ar *AntiReplay) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the body to get counter
		var req model.CombinedRequest
		body, parsedReq, err := parseCombinedRequest(r)
		if err != nil {
			response.BadRequest(w, response.ErrInternalError, "Invalid request body")
			return
		}
		req = *parsedReq

		deviceID := req.SideA.DeviceID
		newCtr := req.SideA.Ctr
		key := fmt.Sprintf("ctr:%s", deviceID)

		// Execute Lua script atomically
		result, err := ar.script.Run(r.Context(), ar.redis,
			[]string{key},
			newCtr, 86400, // 24h TTL
		).Int()

		if err != nil {
			slog.Error("Anti-replay Redis error, allowing request", "error", err)
			// Fail-open: if Redis is down, let the DB constraint catch replays
		} else if result == 0 {
			slog.Warn("Replay attack detected",
				"device_id", deviceID,
				"ctr", newCtr,
			)
			response.BadRequest(w, response.ErrInvalidCounter,
				fmt.Sprintf("Counter %d rejected — replay attempt", newCtr))
			return
		}

		// Store parsed request in context for downstream middleware
		ctx := context.WithValue(r.Context(), combinedRequestKey, &req)
		ctx = context.WithValue(ctx, rawBodyKey, body)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CheckCounter checks a counter value without the full middleware chain (for testing)
func (ar *AntiReplay) CheckCounter(ctx context.Context, deviceID string, ctr int64) (bool, error) {
	key := fmt.Sprintf("ctr:%s", deviceID)
	result, err := ar.script.Run(ctx, ar.redis, []string{key}, ctr, 86400).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// === Context keys and helpers ===

type contextKey string

const (
	combinedRequestKey contextKey = "combinedRequest"
	rawBodyKey         contextKey = "rawBody"
	deviceAKey         contextKey = "deviceA"
	deviceBKey         contextKey = "deviceB"
)

// GetCombinedRequest extracts the parsed request from context
func GetCombinedRequest(ctx context.Context) *model.CombinedRequest {
	if val, ok := ctx.Value(combinedRequestKey).(*model.CombinedRequest); ok {
		return val
	}
	return nil
}

// GetDeviceA extracts device A from context
func GetDeviceA(ctx context.Context) *model.Device {
	if val, ok := ctx.Value(deviceAKey).(*model.Device); ok {
		return val
	}
	return nil
}

// SetDeviceA stores device A in context
func SetDeviceA(ctx context.Context, device *model.Device) context.Context {
	return context.WithValue(ctx, deviceAKey, device)
}

// GetDeviceB extracts device B from context
func GetDeviceB(ctx context.Context) *model.Device {
	if val, ok := ctx.Value(deviceBKey).(*model.Device); ok {
		return val
	}
	return nil
}

// parseCombinedRequest reads and parses body, returns raw bytes and parsed struct
func parseCombinedRequest(r *http.Request) ([]byte, *model.CombinedRequest, error) {
	// Check if already parsed
	if req := GetCombinedRequest(r.Context()); req != nil {
		return nil, req, nil
	}

	// Read body
	var buf [16384]byte
	n, _ := r.Body.Read(buf[:])
	body := buf[:n]

	var req model.CombinedRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, nil, err
	}

	return body, &req, nil
}

// ParseCtrFromHeader reads counter from X-Counter header (alternative transport)
func ParseCtrFromHeader(r *http.Request) (int64, error) {
	ctrStr := r.Header.Get("X-Counter")
	if ctrStr == "" {
		return -1, fmt.Errorf("X-Counter header not found")
	}
	return strconv.ParseInt(ctrStr, 10, 64)
}
