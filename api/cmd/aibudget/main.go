// Command aibudget reads shared provider budget metrics. It runs no model or database work.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"classgame-api/internal/config"
	"classgame-api/internal/platform/genai/redisbudget"
	"classgame-api/internal/platform/redis"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 1 {
		return fmt.Errorf("aibudget is read-only and accepts no arguments")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, err := redis.Connect(ctx, redis.Options{URL: cfg.RedisURL, CACertPath: cfg.RedisCACertPath, CACertPEM: cfg.RedisCACertPEM, PingTimeout: 2 * time.Second})
	if err != nil {
		return fmt.Errorf("budget Redis is unavailable")
	}
	defer connection.Disconnect(context.Background())
	budget, err := redisbudget.New(connection.Client(), redisbudget.Options{Scope: cfg.GenAISharedBudgetScope, Concurrent: int64(cfg.GenAISharedConcurrent), RequestsPerWindow: int64(cfg.GenAISharedRequestsPerMinute), OutputTokensPerWindow: int64(cfg.GenAISharedOutputPerMinute), Window: time.Minute, Lease: time.Duration(cfg.GenAISharedLeaseSeconds) * time.Second})
	if err != nil {
		return err
	}
	defer budget.Close()
	snapshot, err := budget.Snapshot(ctx)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"scope": cfg.GenAISharedBudgetScope, "concurrent_limit": cfg.GenAISharedConcurrent, "requests_per_minute": cfg.GenAISharedRequestsPerMinute, "output_tokens_per_minute": cfg.GenAISharedOutputPerMinute, "observed": snapshot})
}
