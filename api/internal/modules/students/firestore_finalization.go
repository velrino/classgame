package students

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (repository *FirestoreRepository) ClaimStudentFinalization(ctx context.Context, job LearningJob, token string, now, until time.Time) (StudentMissionAttemptDocument, UserDocument, bool, error) {
	var attempt StudentMissionAttemptDocument
	var user UserDocument
	claimed := false
	if token == "" || !until.After(now) {
		return attempt, user, false, fmt.Errorf("invalid finalization lease")
	}
	ref := repository.client.Collection(studentAttemptsCollectionName).Doc(job.AttemptID)
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		claimed = false
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return ErrStudentAttemptNotFound
		}
		if err != nil {
			return fmt.Errorf("read finalization attempt: %w", err)
		}
		if err := snapshot.DataTo(&attempt); err != nil {
			return err
		}
		if attempt.ID == "" {
			attempt.ID = ref.ID
		}
		if attempt.StudentID != job.StudentID {
			return ErrStudentMissionMismatch
		}
		if !FinalizationJobMatches(attempt, job) || !FinalizationEligible(attempt) {
			return nil
		}
		if !FinalizationDue(attempt, now) {
			return ErrFinalizationBusy
		}
		userSnapshot, err := tx.Get(repository.client.Collection(usersCollectionName).Doc(job.StudentID))
		if status.Code(err) == codes.NotFound {
			return ErrStudentNotFound
		}
		if err != nil {
			return fmt.Errorf("read finalization student: %w", err)
		}
		if err := userSnapshot.DataTo(&user); err != nil {
			return err
		}
		if user.ID == "" {
			user.ID = userSnapshot.Ref.ID
		}
		if user.ID != job.StudentID {
			return ErrStudentMissionMismatch
		}
		attempt.FinalizationToken = token
		attempt.FinalizationLeaseUntil = until
		attempt.TraceState = "processing"
		if err := tx.Set(ref, map[string]any{"finalization_token": token, "finalization_lease_until": until, "trace_state": "processing"}, firestore.MergeAll); err != nil {
			return err
		}
		claimed = true
		return nil
	})
	return attempt, user, claimed, err
}

func (repository *FirestoreRepository) FailStudentFinalization(ctx context.Context, fence FinalizationFence, now time.Time) error {
	ref := repository.client.Collection(studentAttemptsCollectionName).Doc(fence.AttemptID)
	return repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil
		}
		if err != nil {
			return err
		}
		var attempt StudentMissionAttemptDocument
		if err := snapshot.DataTo(&attempt); err != nil {
			return err
		}
		if attempt.ID == "" {
			attempt.ID = ref.ID
		}
		if !MatchesFinalizationFence(attempt, fence) {
			return nil
		}
		return tx.Set(ref, map[string]any{"trace_state": "failed", "finalization_token": "", "finalization_lease_until": time.Time{}, "updated_at": now}, firestore.MergeAll)
	})
}
