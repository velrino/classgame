// Package redisbudget applies one provider budget across API and worker processes.
package redisbudget

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"classgame-api/internal/platform/genai"
)

// TIME is authoritative; host clock skew cannot free another process's capacity.
// All keys share a Redis hash slot. Rejected requests consume no window budget.
var acquire = redis.NewScript(`
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
local lease = tonumber(ARGV[2])
local window = tonumber(ARGV[3])
local limit = tonumber(ARGV[4])
local requests = tonumber(ARGV[5])
local tokens = tonumber(ARGV[6])
local cost = tonumber(ARGV[7])
local policy = table.concat({ARGV[2], ARGV[3], ARGV[4], ARGV[5], ARGV[6]}, ':')
local existing = redis.call('GET', KEYS[3])
if existing and existing ~= policy then return {-2, 0, 0, 0} end
redis.call('PSETEX', KEYS[3], 2 * math.max(lease, window), policy)
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now)
local active = redis.call('ZCARD', KEYS[1])
local epoch = math.floor(now / window)
local used = 0
local reserved = 0
if tonumber(redis.call('HGET', KEYS[2], 'epoch')) == epoch then
  used = tonumber(redis.call('HGET', KEYS[2], 'requests')) or 0
  reserved = tonumber(redis.call('HGET', KEYS[2], 'tokens')) or 0
end
if active >= limit then return {0, active, used, reserved} end
if used + 1 > requests or reserved + cost > tokens then return {-1, active, used, reserved} end
redis.call('HSET', KEYS[2], 'epoch', epoch, 'requests', used + 1, 'tokens', reserved + cost)
redis.call('PEXPIRE', KEYS[2], 2 * window)
redis.call('ZADD', KEYS[1], now + lease, ARGV[1])
redis.call('PEXPIRE', KEYS[1], 2 * lease)
return {1, active + 1, used + 1, reserved + cost}
`)

var release = redis.NewScript(`return redis.call('ZREM', KEYS[1], ARGV[1])`)
var snapshot = redis.NewScript(`
local clock = redis.call('TIME')
local now = tonumber(clock[1]) * 1000 + math.floor(tonumber(clock[2]) / 1000)
local active = redis.call('ZCOUNT', KEYS[1], '(' .. now, '+inf')
if tonumber(redis.call('HGET', KEYS[2], 'epoch')) ~= math.floor(now / tonumber(ARGV[1])) then
  return {active, 0, 0}
end
return {active, tonumber(redis.call('HGET', KEYS[2], 'requests')) or 0, tonumber(redis.call('HGET', KEYS[2], 'tokens')) or 0}
`)

type Options struct {
	Scope                 string
	Concurrent            int64
	RequestsPerWindow     int64
	OutputTokensPerWindow int64
	Window                time.Duration
	Lease                 time.Duration
}

type Budget struct {
	client  *redis.Client
	options Options
	keys    []string
}

func New(client *redis.Client, options Options) (*Budget, error) {
	if client == nil || strings.TrimSpace(options.Scope) == "" || len(options.Scope) > 120 || options.Concurrent < 1 || options.Concurrent > 1000 || options.RequestsPerWindow < 1 || options.RequestsPerWindow > 1000000 || options.OutputTokensPerWindow < 1 || options.OutputTokensPerWindow > 1000000000 || options.Window < 100*time.Millisecond || options.Window > time.Hour || options.Lease < 100*time.Millisecond || options.Lease > 5*time.Minute {
		return nil, errors.New("invalid shared provider budget settings")
	}
	digest := sha256.Sum256([]byte(strings.TrimSpace(options.Scope)))
	prefix := "classgame:provider_budget:{" + hex.EncodeToString(digest[:]) + "}"
	// A separate pool keeps admission/release timeouts bounded even if the cache
	// client has longer I/O or retry settings. Connection secrets are reused in memory.
	connection := *client.Options()
	connection.ContextTimeoutEnabled = true
	connection.DialTimeout = 500 * time.Millisecond
	connection.ReadTimeout = 500 * time.Millisecond
	connection.WriteTimeout = 500 * time.Millisecond
	connection.PoolTimeout = 500 * time.Millisecond
	connection.MaxRetries = -1
	connection.MinIdleConns = 0
	connection.PoolSize = 16
	return &Budget{client: redis.NewClient(&connection), options: options, keys: []string{prefix + ":leases", prefix + ":window", prefix + ":policy"}}, nil
}

func (budget *Budget) Close() error { return budget.client.Close() }

func (budget *Budget) Acquire(ctx context.Context, outputTokens int32) (genai.ProviderLease, error) {
	if outputTokens < 1 || int64(outputTokens) > budget.options.OutputTokensPerWindow {
		return nil, genai.ErrSharedBudget
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return nil, genai.ErrBudgetUnavailable
	}
	owner := hex.EncodeToString(token)
	values, err := acquire.Run(ctx, budget.client, budget.keys, owner, budget.options.Lease.Milliseconds(), budget.options.Window.Milliseconds(), budget.options.Concurrent, budget.options.RequestsPerWindow, budget.options.OutputTokensPerWindow, outputTokens).Int64Slice()
	if err != nil || len(values) != 4 {
		return nil, genai.ErrBudgetUnavailable
	}
	if values[0] == -2 {
		return nil, genai.ErrBudgetPolicy
	}
	if values[0] != 1 {
		return nil, genai.ErrSharedBudget
	}
	return &lease{budget: budget, owner: owner}, nil
}

type lease struct {
	budget *Budget
	owner  string
}

func (lease *lease) Release(ctx context.Context) error {
	if err := release.Run(ctx, lease.budget.client, lease.budget.keys[:1], lease.owner).Err(); err != nil {
		return genai.ErrBudgetUnavailable
	}
	return nil
}

// Snapshot is read-only and suitable for metrics or an operator's inspection.
type Snapshot struct {
	Active               int64 `json:"active_leases"`
	Requests             int64 `json:"window_requests"`
	ReservedOutputTokens int64 `json:"window_output_tokens"`
}

func (budget *Budget) Snapshot(ctx context.Context) (Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	values, err := snapshot.Run(ctx, budget.client, budget.keys[:2], budget.options.Window.Milliseconds()).Int64Slice()
	if err != nil || len(values) != 3 {
		return Snapshot{}, genai.ErrBudgetUnavailable
	}
	return Snapshot{Active: values[0], Requests: values[1], ReservedOutputTokens: values[2]}, nil
}
