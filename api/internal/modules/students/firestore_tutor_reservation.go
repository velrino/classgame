package students

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ReserveTutorGeneration claims one event before paid work. The attempt counter
// counts durable admissions, including uncertain failed/crashed provider calls.
func (repository *FirestoreRepository) ReserveTutorGeneration(ctx context.Context, input SaveStudentTutorTurnInput, owner string, now, until time.Time) (TutorGenerationReservation, error) {
	result := TutorGenerationReservation{}
	if owner == "" || !until.After(now) {
		return result, fmt.Errorf("invalid tutor reservation lease")
	}
	attemptRef := repository.client.Collection(studentAttemptsCollectionName).Doc(input.ExpectedAttemptID)
	eventRef := attemptRef.Collection("tutor_events").Doc(tutorTurnID(input.EventID))
	err := repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		result = TutorGenerationReservation{}
		userSnapshot, err := tx.Get(repository.client.Collection(usersCollectionName).Doc(input.Identity.ProviderUserID))
		if err != nil {
			return err
		}
		var user UserDocument
		if err := userSnapshot.DataTo(&user); err != nil {
			return err
		}
		attemptSnapshot, err := tx.Get(attemptRef)
		if status.Code(err) == codes.NotFound {
			return ErrStudentAttemptNotFound
		}
		if err != nil {
			return err
		}
		var attempt StudentMissionAttemptDocument
		if err := attemptSnapshot.DataTo(&attempt); err != nil {
			return err
		}
		if attempt.ID == "" {
			attempt.ID = attemptRef.ID
		}
		if !sameStudentClass(user.Student, attempt) {
			return ErrStudentMissionMismatch
		}
		if err := ValidateTutorTurnScope(input, user.Activity.CurrentGame, attempt); err != nil {
			return err
		}
		eventSnapshot, eventErr := tx.Get(eventRef)
		if eventErr == nil {
			var event TutorEventDocument
			if err := eventSnapshot.DataTo(&event); err != nil {
				return err
			}
			turn, complete, err := TutorReplayDecision(event, input, now)
			if err != nil {
				return err
			}
			if complete {
				result = TutorGenerationReservation{Completed: true, Turn: turn}
				return nil
			}
		} else if status.Code(eventErr) != codes.NotFound {
			return eventErr
		}
		if attempt.Status == "completed" {
			return ErrStudentAttemptClosed
		}
		attempt = EffectiveTutorEvidence(attempt)
		if attempt.LegacyHistoryIncomplete {
			return ErrTutorHistoryIncomplete
		}
		if !CanGenerateTutorTurn(attempt) {
			return ErrStudentTutorTurnLimit
		}
		attempt.GeneratedTurnCount++
		event := TutorEventDocument{StudentID: input.Identity.ProviderUserID, EventID: input.EventID, Fingerprint: TutorEventFingerprint(input), State: "processing", OwnerToken: owner, LeaseUntil: until, CreatedAt: now}
		if err := tx.Set(eventRef, event); err != nil {
			return err
		}
		return tx.Set(attemptRef, map[string]any{"evidence_schema_version": attempt.EvidenceSchemaVersion, "accepted_answers": attempt.AcceptedAnswers, "generated_turn_count": attempt.GeneratedTurnCount, "tutor_event_sequence": attempt.TutorEventSequence, "legacy_history_incomplete": attempt.LegacyHistoryIncomplete}, firestore.MergeAll)
	})
	return result, err
}
func (repository *FirestoreRepository) FailTutorGeneration(ctx context.Context, input SaveStudentTutorTurnInput, now time.Time) error {
	ref := repository.client.Collection(studentAttemptsCollectionName).Doc(input.ExpectedAttemptID).Collection("tutor_events").Doc(tutorTurnID(input.EventID))
	return repository.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil
		}
		if err != nil {
			return err
		}
		var event TutorEventDocument
		if err := snapshot.DataTo(&event); err != nil {
			return err
		}
		// Expiry alone does not permit an old owner to reset a successor's lease.
		if event.OwnerToken != input.ReservationOwner || event.State != "processing" || event.Fingerprint != TutorEventFingerprint(input) {
			return nil
		}
		return tx.Set(ref, map[string]any{"state": "failed", "owner_token": "", "lease_until": time.Time{}, "updated_at": now}, firestore.MergeAll)
	})
}
