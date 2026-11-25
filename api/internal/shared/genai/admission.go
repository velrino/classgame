package genai

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

var (
	ErrSharedBudget      = errors.New("shared provider budget is full")
	ErrBudgetUnavailable = errors.New("shared provider budget is unavailable")
	ErrBudgetPolicy      = errors.New("shared provider budget settings differ between instances")
)

// ProviderAdmission is owned by the provider client. Every model workflow uses it.
type ProviderAdmission interface {
	Acquire(context.Context, int32) (ProviderLease, error)
}

type ProviderLease interface {
	Release(context.Context) error
}

// RunAdmitted holds shared capacity only while provider work runs. Window usage
// is not refunded: a timeout or lost response may still have cost money.
func RunAdmitted(ctx context.Context, budget ProviderAdmission, outputTokens int32, work func() error) error {
	if budget == nil {
		return ErrBudgetUnavailable
	}
	lease, err := budget.Acquire(ctx, outputTokens)
	if err != nil {
		slog.WarnContext(ctx, "genai shared admission rejected", "reason", err)
		return err
	}
	if lease == nil {
		return ErrBudgetUnavailable
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		if err := lease.Release(releaseCtx); err != nil {
			slog.WarnContext(ctx, "genai shared lease release failed", "reason", "shared store unavailable; lease will expire")
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	return work()
}
