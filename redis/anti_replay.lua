-- ═══════════════════════════════════════════════
-- Anti-Replay Redis Lua Script
-- ═══════════════════════════════════════════════
-- KEYS[1] = "ctr:{deviceId}"
-- ARGV[1] = new counter value
-- ARGV[2] = TTL in seconds (optional, default 86400 = 24h)
--
-- Returns:
--   1  → counter accepted (new > stored)
--   0  → counter rejected (replay attempt)
-- ═══════════════════════════════════════════════

local key = KEYS[1]
local newCtr = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2]) or 86400

local storedCtr = tonumber(redis.call('GET', key))

if storedCtr == nil then
    -- First request from this device — accept
    redis.call('SETEX', key, ttl, newCtr)
    return 1
end

if newCtr > storedCtr then
    -- Valid: counter is strictly increasing
    redis.call('SETEX', key, ttl, newCtr)
    return 1
else
    -- Replay attempt: counter not increasing
    return 0
end
