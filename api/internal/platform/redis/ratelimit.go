package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"classgame-api/internal/ratelimit"
)

var (
	ErrMissingClient = errors.New("rate limiter redis client is required")
	ErrInvalidLimit  = errors.New("rate limiter limit must be greater than zero")
	ErrInvalidWindow = errors.New("rate limiter window must be greater than zero")
)

const defaultPrefix = "classgame:rate_limit"

var fixedWindowScript = goredis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[2])
end

local ttl = redis.call("PTTL", KEYS[1])
if ttl < 0 then
  redis.call("PEXPIRE", KEYS[1], ARGV[2])
  ttl = tonumber(ARGV[2])
end

local allowed = 0
if count <= tonumber(ARGV[1]) then
  allowed = 1
end

return {allowed, count, ttl}
`)

type FixedWindowOptions struct {
	Limit  int64
	Prefix string
	Window time.Duration
}

type RedisFixedWindow struct {
	client goredis.UniversalClient
	limit  int64
	prefix string
	window time.Duration
}

func NewFixedWindow(
	client goredis.UniversalClient,
	options FixedWindowOptions,
) (*RedisFixedWindow, error) {
	if client == nil {
		return nil, ErrMissingClient
	}
	if options.Limit <= 0 {
		return nil, ErrInvalidLimit
	}
	if options.Window <= 0 {
		return nil, ErrInvalidWindow
	}

	prefix := strings.TrimSpace(options.Prefix)
	if prefix == "" {
		prefix = defaultPrefix
	}

	return &RedisFixedWindow{
		client: client,
		limit:  options.Limit,
		prefix: prefix,
		window: options.Window,
	}, nil
}

func (limiter *RedisFixedWindow) Allow(
	ctx context.Context,
	identifier string,
) (ratelimit.Decision, error) {
	if limiter == nil || limiter.client == nil {
		return ratelimit.Decision{}, ErrMissingClient
	}

	values, err := fixedWindowScript.Run(
		ctx,
		limiter.client,
		[]string{limiter.key(identifier)},
		limiter.limit,
		limiter.window.Milliseconds(),
	).Int64Slice()
	if err != nil {
		return ratelimit.Decision{}, fmt.Errorf("apply redis rate limit: %w", err)
	}
	if len(values) != 3 {
		return ratelimit.Decision{}, fmt.Errorf("apply redis rate limit: unexpected result length %d", len(values))
	}

	remaining := limiter.limit - values[1]
	if remaining < 0 {
		remaining = 0
	}

	decision := ratelimit.Decision{
		Allowed:   values[0] == 1,
		Limit:     limiter.limit,
		Remaining: remaining,
	}
	if !decision.Allowed && values[2] > 0 {
		decision.RetryAfter = time.Duration(values[2]) * time.Millisecond
	}

	return decision, nil
}

func (limiter *RedisFixedWindow) key(identifier string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(identifier)))

	return limiter.prefix + ":" + hex.EncodeToString(digest[:])
}
