package students

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListPendingFinalizations scans at most limit rows by document ID and uses a
// narrow projection. It needs no composite index and never scans the whole DB.
func (repository *FirestoreRepository) ListPendingFinalizations(ctx context.Context, after string, limit int, now time.Time) (RecoveryPage, error) {
	page := RecoveryPage{}
	if limit < 1 || limit > 100 || len(after) > 120 || strings.Contains(after, "/") {
		return page, errors.New("recovery requires limit 1..100 and a bounded document cursor")
	}
	query := repository.client.Collection(studentAttemptsCollectionName).OrderBy(firestore.DocumentID, firestore.Asc).Select("id", "student_id", "class_id", "published_revision", "status", "trace_state", "trace.finalized", "evidence_schema_version", "evidence_revision", "snapshot.revision", "finalization_lease_until", "finalization_dispatch_generation").Limit(limit + 1)
	if after != "" {
		query = query.StartAfter(after)
	}
	rows, err := query.Documents(ctx).GetAll()
	if err != nil {
		return page, fmt.Errorf("list finalization recovery page: %w", err)
	}
	if len(rows) > limit {
		rows = rows[:limit]
		page.NextCursor = rows[len(rows)-1].Ref.ID
	}
	page.Scanned = len(rows)
	for _, row := range rows {
		var attempt StudentMissionAttemptDocument
		if err := row.DataTo(&attempt); err != nil {
			return page, fmt.Errorf("decode recovery candidate: %w", err)
		}
		attempt.ID = row.Ref.ID
		if FinalizationDue(attempt, now) {
			page.Attempts = append(page.Attempts, attempt)
		}
	}
	return page, nil
}
func (repository *FirestoreRepository) PrepareFinalizationRedrive(ctx context.Context, expected LearningJob, now time.Time) (StudentMissionAttemptDocument, bool, error) {
	var attempt StudentMissionAttemptDocument
	changed := false
	ref := repository.client.Collection(studentAttemptsCollectionName).Doc(expected.AttemptID)
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		changed = false
		attempt = StudentMissionAttemptDocument{}
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil
		}
		if err != nil {
			return err
		}
		if err := snapshot.DataTo(&attempt); err != nil {
			return err
		}
		if attempt.ID == "" {
			attempt.ID = ref.ID
		}
		if !FinalizationJobMatches(attempt, expected) || !FinalizationDue(attempt, now) {
			return nil
		}
		attempt.FinalizationDispatchGeneration++
		attempt.TraceState = "pending"
		attempt.FinalizationToken = ""
		attempt.FinalizationLeaseUntil = time.Time{}
		attempt.FinalizeRequestedAt = now
		if err := tx.Set(ref, map[string]any{"finalization_dispatch_generation": attempt.FinalizationDispatchGeneration, "trace_state": "pending", "finalization_token": "", "finalization_lease_until": time.Time{}, "finalize_requested_at": now}, firestore.MergeAll); err != nil {
			return err
		}
		changed = true
		return nil
	})
	return attempt, changed, err
}
