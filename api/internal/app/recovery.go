package app

import (
	"context"
	"log/slog"
	"time"
)

// StartBackgroundRecovery scans one bounded page per minute in worker roles.
// Durable pending markers and domain version/lease checks remain authoritative.
func StartBackgroundRecovery(ctx context.Context, sweep func(context.Context, string, int) (string, error)) func() {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		timer := time.NewTimer(0)
		defer timer.Stop()
		after := ""
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
			runContext, stop := context.WithTimeout(ctx, 30*time.Second)
			next, err := sweep(runContext, after, 50)
			stop()
			if err != nil {
				slog.WarnContext(ctx, "background recovery page failed")
			} else {
				after = next
			}
			timer.Reset(time.Minute)
		}
	}()
	return func() { cancel(); <-done }
}
