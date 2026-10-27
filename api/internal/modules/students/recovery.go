package students

import (
	"context"
	"errors"
	"time"
)

type RecoveryPage struct {
	Attempts   []StudentMissionAttemptDocument
	NextCursor string
	Scanned    int
}
type RecoveryReport struct {
	Scanned    int    `json:"scanned"`
	Candidates int    `json:"candidates"`
	Enqueued   int    `json:"enqueued"`
	Failed     int    `json:"failed"`
	NextCursor string `json:"next_cursor,omitempty"`
}
type RecoveryStore interface {
	ListPendingFinalizations(context.Context, string, int, time.Time) (RecoveryPage, error)
	PrepareFinalizationRedrive(context.Context, LearningJob, time.Time) (StudentMissionAttemptDocument, bool, error)
}

// RecoverPendingFinalizations retries stable delivery IDs. Callers advance the
// cursor over bounded pages; this method never generates model content itself.
func (service *Service) RecoverPendingFinalizations(ctx context.Context, after string, limit int) (RecoveryReport, error) {
	return service.recoverFinalizations(ctx, after, limit, false)
}

// RedrivePendingFinalizations starts a new dispatch generation for an operator's
// explicit retry of exhausted/archived delivery IDs. Normal sweeps do not do this.
func (service *Service) RedrivePendingFinalizations(ctx context.Context, after string, limit int) (RecoveryReport, error) {
	return service.recoverFinalizations(ctx, after, limit, true)
}
func (service *Service) recoverFinalizations(ctx context.Context, after string, limit int, redrive bool) (RecoveryReport, error) {
	report := RecoveryReport{}
	if service.recovery == nil || service.learningTaskEnqueuer == nil {
		return report, errors.New("learning recovery store and task enqueuer are required")
	}
	page, err := service.recovery.ListPendingFinalizations(ctx, after, limit, time.Now())
	if err != nil {
		return report, err
	}
	report.Scanned = page.Scanned
	report.Candidates = len(page.Attempts)
	report.NextCursor = page.NextCursor
	for _, attempt := range page.Attempts {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		if redrive {
			current, changed, err := service.recovery.PrepareFinalizationRedrive(ctx, FinalizationJob(attempt), time.Now())
			if err != nil {
				report.Failed++
				continue
			}
			if !changed {
				continue
			}
			attempt = current
		}
		if err := service.learningTaskEnqueuer.EnqueueFinalization(ctx, attempt); err != nil {
			report.Failed++
		} else {
			report.Enqueued++
		}
	}
	if report.Failed > 0 {
		return report, errors.New("some learning finalizations could not be enqueued")
	}
	return report, nil
}
