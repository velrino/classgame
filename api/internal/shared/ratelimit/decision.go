package ratelimit

import (
	"context"
	"time"
)

type Decision struct {
	Allowed    bool
	Limit      int64
	Remaining  int64
	RetryAfter time.Duration
}

type Limiter interface {
	Allow(context.Context, string) (Decision, error)
}
