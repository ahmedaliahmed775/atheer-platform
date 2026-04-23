// Modified for v3.0 Document Alignment
package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/atheer-payment/atheer-platform/internal/model"
	"github.com/atheer-payment/atheer-platform/pkg/response"
	"github.com/redis/go-redis/v9"
)

type AntiReplay struct {
	redis  *redis.Client
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
		var req model.PayerTlvPacket
		var buf [16384]byte
		n, _ := r.Body.Read(buf[:])
		body := buf[:n]

		if err := json.Unmarshal(body, &req); err != nil {
			response.BadRequest(w, response.ErrInternalError, "Invalid request body")
			return
		}

		publicID := req.PublicID
		newCtr := req.Counter
		key := fmt.Sprintf("ctr:%s", publicID)

		result, err := ar.script.Run(r.Context(), ar.redis,
			[]string{key},
			newCtr, 86400,
		).Int()

		if err != nil {
			slog.Error("Anti-replay Redis error — REJECTING request (fail-closed)", "error", err)
			response.ServiceUnavailable(w, "ERR_ANTI_REPLAY", "Security service unavailable — retry later")
			return
		} else if result == 0 {
			slog.Warn("Replay attack detected", "public_id", publicID, "ctr", newCtr)
			response.BadRequest(w, response.ErrInternalError, fmt.Sprintf("Counter %d rejected", newCtr))
			return
		}

		ctx := SetPayerPacket(r.Context(), &req)
		
		// Set deviceAKey to avoid breaking older handlers if any.
		// Wait, no need since we updated the handlers (or we will).
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
