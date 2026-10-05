package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/webcore-go/webcore/port"
)

// jam dari Redis (TIME), bukan dari pod, supaya selisih jam antar pod tidak menggeser window
var skripRateLimit = redis.NewScript(`
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local start = now - (now % window)
local kcur = KEYS[1] .. ':' .. start
local kprev = KEYS[1] .. ':' .. (start - window)
local cur = tonumber(redis.call('GET', kcur) or '0')
local prev = tonumber(redis.call('GET', kprev) or '0')
local est = prev * (window - (now - start)) / window + cur
if est + 1 > limit then
	return {0, 0, start + window}
end
redis.call('INCR', kcur)
redis.call('PEXPIRE', kcur, window * 2)
return {1, math.floor(limit - est - 1), start + window}
`)

var _ port.IRateLimitStore = (*Redis)(nil)

func (r *Redis) RateLimit(ctx context.Context, key string, limit int64, window time.Duration) (port.RateLimitResult, error) {
	hasil, err := skripRateLimit.Run(ctx, r.Client, []string{key}, window.Milliseconds(), limit).Int64Slice()
	if err != nil {
		return port.RateLimitResult{}, err
	}

	return port.RateLimitResult{
		Allowed:   hasil[0] == 1,
		Remaining: hasil[1],
		ResetAt:   time.UnixMilli(hasil[2]),
	}, nil
}
