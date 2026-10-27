package ratelimit

import (
	"context"
	"errors"
	"testing"
)

func TestEvaluateReturnsFirstDeniedRule(t *testing.T) {
	allowed := &fakeLimiter{decision: Decision{Allowed: true, Limit: 12, Remaining: 11}}
	denied := &fakeLimiter{decision: Decision{Allowed: false, Limit: 60, Remaining: 0}}
	notReached := &fakeLimiter{decision: Decision{Allowed: true}}

	policy, err := Evaluate(context.Background(),
		Rule{Limiter: allowed, Identifier: "student:1"},
		Rule{Limiter: denied, Identifier: "student:1"},
		Rule{Limiter: notReached, Identifier: "ip:1"},
	)
	if err != nil {
		t.Fatalf("evaluate policy: %v", err)
	}
	if !policy.Evaluated || policy.Decision.Allowed || policy.Decision.Limit != 60 {
		t.Fatalf("policy = %#v", policy)
	}
	if notReached.calls != 0 {
		t.Fatal("rules after a denial should not be consumed")
	}
}

func TestEvaluatePropagatesLimiterFailure(t *testing.T) {
	want := errors.New("redis unavailable")
	_, err := Evaluate(context.Background(), Rule{
		Limiter: &fakeLimiter{err: want},
	})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

type fakeLimiter struct {
	calls    int
	decision Decision
	err      error
}

func (limiter *fakeLimiter) Allow(context.Context, string) (Decision, error) {
	limiter.calls++
	return limiter.decision, limiter.err
}
